package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/ps_package"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"sync"
)

func (s *agentServer) Download(ctx context.Context, req *api.DownloadRequest) (*api.DownloadResponse, error) {

	res := &ps_package.GetPresignedUrlResponse{}

	var err error

	responseType := api.DownloadResponse_PRESIGNED_URL

	switch req.Type {
	case api.DownloadRequest_PACKAGE:
		log.Debug("download request for package")

		// Download single Package
		requestData := req.GetPackage()

		// Need to get a presigned URL for the package.
		// This can return an array of results in case a package has multiple source files.
		client := s.client
		res, err = client.Package.GetPresignedUrl(ctx, requestData.PackageId, true)

		if !requestData.GetPresignedUrl {
			log.Debug("Downloading the package.")
			go func() {
				// Iterate over the files in a package and download serially
				// TODO: can optimize by concurrency
				for _, f := range res.Files {

					downloaderImpl := shared.NewDownloader(s, s.client)
					_, err = downloaderImpl.DownloadFileFromPresignedUrl(ctx, f.URL, f.Name, requestData.PackageId)
					if err != nil {
						log.Errorf("Download failed: %v", err)
					}
				}

			}()
		}

	case api.DownloadRequest_DATASET:

		// Request Data Should contain dataset-id
		requestData := req.GetDataset()

		//
		client := s.client
		manifestResponse, err := client.Dataset.GetManifest(ctx, requestData.DatasetId)
		if err != nil {
			log.Errorf("Download failed: %v", err)

			return nil, err
		}

		// Create folder (and include hidden .pennsieve folder for manifest)
		err = os.MkdirAll(filepath.Join(requestData.TargetFolder, ".pennsieve"), os.ModePerm)
		if err != nil {
			log.Errorf("Failed to create target path: %v", err)
			return nil, err
		}

		// Download Manifest to hidden .pennsieve folder in target path
		manifestLocation := filepath.Join(requestData.TargetFolder, ".pennsieve", "manifest.json")

		downloaderImpl := shared.NewDownloader(s, s.client)

		_, err = downloaderImpl.DownloadFileFromPresignedUrl(ctx, manifestResponse.URL, manifestLocation, uuid.New().String())
		if err != nil {
			log.Errorf("Download failed: %v", err)
		}

		// Now read in manifest
		manifestFile, err := os.Open(filepath.FromSlash(manifestLocation))
		if err != nil {
			fmt.Printf("failed to open manifest file: %s, error: %v", manifestLocation, err)
			return nil, err
		}
		defer func(manifestFile *os.File) {
			err := manifestFile.Close()
			if err != nil {
				log.Warnf("Failed to close manifest file: %s, error: %v", manifestLocation, err)
			}
		}(manifestFile)

		jsonData, err := io.ReadAll(manifestFile)
		if err != nil {
			fmt.Printf("failed to read json file, error: %v", err)
			return nil, err
		}

		data := models.WorkspaceManifest{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			fmt.Printf("failed to unmarshal json file, error: %v", err)
			return nil, err
		}

		// Go routines for downloading data in parallel
		nrWorkers := 5
		walker := make(chan models.ManifestDTO, nrWorkers)
		results := make(chan int, nrWorkers)
		var downloadWg sync.WaitGroup

		go func() {
			defer close(walker)
			for _, file := range data.Files {
				walker <- file
			}
		}()

		for w := 0; w < nrWorkers; w++ {
			downloadWg.Add(1)
			log.Println("Starting download worker: ", w)

			w := w
			go func() {
				defer func() {
					log.Println("Closing download worker: ", w)
					downloadWg.Done()
				}()

				downloaderImpl.DownloadWorker(ctx, w, walker, results, requestData.TargetFolder)

			}()

		}

		downloadWg.Wait()

	}

	resp := &api.DownloadResponse{
		Type:   responseType,
		Status: "Success",
		Url:    []string{""},
	}

	return resp, nil
}
