package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/pkg/shared"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/download"
	log "github.com/sirupsen/logrus"
)

func (s *agentServer) Download(ctx context.Context, req *api.DownloadRequest) (*api.DownloadResponse, error) {
	client, err := s.PennsieveClient()
	if err != nil {
		return nil, err
	}

	switch req.Type {
	case api.DownloadRequest_PACKAGE:
		log.Debug("download request for package")
		requestData := req.GetPackage()
		if requestData.DatasetId == "" {
			return s.downloadPackageFromAPI(ctx, client, requestData)
		}
		return s.downloadPackage(ctx, client, requestData)

	case api.DownloadRequest_DATASET:
		return s.downloadDataset(ctx, client, req.GetDataset())
	}
	return nil, fmt.Errorf("unknown download type: %v", req.Type)
}

// downloadDataset downloads a dataset, or the folders and packages in
// NodeIds, into TargetFolder through download-service. The first page of
// links is requested before returning, so a selection that can't be
// downloaded fails here; the files download in the background.
func (s *agentServer) downloadDataset(ctx context.Context, client *pennsieve.Client, requestData *api.DownloadDatasetRequest) (*api.DownloadResponse, error) {
	manifestReq := download.ManifestRequest{NodeIds: requestData.NodeIds}
	first, err := client.Download.GetManifestPage(ctx, requestData.DatasetId, manifestReq)
	if err != nil {
		log.Errorf("Download failed: %v", err)
		return nil, err
	}

	if err := os.MkdirAll(requestData.TargetFolder, os.ModePerm); err != nil {
		log.Errorf("Failed to create target path: %v", err)
		return nil, err
	}
	downloader := shared.NewDownloader(s, client)

	// A whole dataset also gets its workspace manifest in the hidden
	// .pennsieve folder, as before.
	if len(requestData.NodeIds) == 0 {
		if err := downloadWorkspaceManifest(ctx, client, &downloader, requestData); err != nil {
			return nil, err
		}
	}

	log.Infof("Downloading %d files (%d bytes) of %s to %s",
		first.Header.Count, first.Header.Size, requestData.DatasetId, requestData.TargetFolder)
	go func() {
		res, err := downloader.DownloadManifest(context.Background(), shared.ManifestDownload{
			DatasetId: requestData.DatasetId,
			Request:   manifestReq,
			Target: func(f download.ManifestFile) string {
				return shared.SafeJoin(requestData.TargetFolder, append(f.Path, f.FileName)...)
			},
		}, first)
		logManifestResult(requestData.DatasetId, res, err)
	}()

	return &api.DownloadResponse{Type: api.DownloadResponse_DOWNLOAD, Status: "Success", Url: []string{""}}, nil
}

func downloadWorkspaceManifest(ctx context.Context, client *pennsieve.Client, downloader shared.Downloader, requestData *api.DownloadDatasetRequest) error {
	manifestResponse, err := client.Dataset.GetManifest(ctx, requestData.DatasetId)
	if err != nil {
		log.Errorf("Download failed: %v", err)
		return err
	}
	if err := os.MkdirAll(filepath.Join(requestData.TargetFolder, ".pennsieve"), os.ModePerm); err != nil {
		log.Errorf("Failed to create target path: %v", err)
		return err
	}
	manifestLocation := filepath.Join(requestData.TargetFolder, ".pennsieve", "manifest.json")
	if _, err := downloader.DownloadFileFromPresignedUrl(ctx, manifestResponse.URL, manifestLocation, uuid.New().String()); err != nil {
		log.Errorf("Download failed: %v", err)
	}
	return nil
}

// downloadPackage downloads a package's files through download-service into
// the agent's folder, or returns their links.
func (s *agentServer) downloadPackage(ctx context.Context, client *pennsieve.Client, requestData *api.DownloadPackageRequest) (*api.DownloadResponse, error) {
	manifestReq := download.ManifestRequest{NodeIds: []string{requestData.PackageId}}
	first, err := client.Download.GetManifestPage(ctx, requestData.DatasetId, manifestReq)
	if err != nil {
		return nil, err
	}

	if requestData.GetPresignedUrl {
		urls := []string{}
		for page := first; ; {
			for _, f := range page.Data {
				urls = append(urls, f.URL)
			}
			if page.Next == "" {
				break
			}
			manifestReq.Cursor = page.Next
			if page, err = client.Download.GetManifestPage(ctx, requestData.DatasetId, manifestReq); err != nil {
				return nil, err
			}
		}
		return &api.DownloadResponse{Type: api.DownloadResponse_PRESIGNED_URL, Status: "Success", Url: urls}, nil
	}

	log.Debug("Downloading the package.")
	downloader := shared.NewDownloader(s, client)
	go func() {
		res, err := downloader.DownloadManifest(context.Background(), shared.ManifestDownload{
			DatasetId: requestData.DatasetId,
			Request:   manifestReq,
			// A package of several files gets a folder of its name (Path).
			Target: func(f download.ManifestFile) string {
				return shared.SafeJoin(".", append(f.Path, f.FileName)...)
			},
		}, first)
		logManifestResult(requestData.PackageId, res, err)
	}()

	return &api.DownloadResponse{Type: api.DownloadResponse_DOWNLOAD, Status: "Success", Url: []string{""}}, nil
}

// downloadPackageFromAPI signs a package's files through pennsieve-api, for
// callers that don't name the package's dataset.
func (s *agentServer) downloadPackageFromAPI(ctx context.Context, client *pennsieve.Client, requestData *api.DownloadPackageRequest) (*api.DownloadResponse, error) {
	res, err := client.Package.GetPresignedUrl(ctx, requestData.PackageId, false)
	if err != nil {
		return nil, err
	}

	if !requestData.GetPresignedUrl {
		log.Debug("Downloading the package.")
		go func() {
			// Iterate over the files in a package and download serially
			for _, f := range res.Files {
				downloaderImpl := shared.NewDownloader(s, client)
				_, err = downloaderImpl.DownloadFileFromPresignedUrl(ctx, f.URL, f.Name, requestData.PackageId)
				if err != nil {
					log.Errorf("Download failed: %v", err)
				}
			}
		}()
	}

	return &api.DownloadResponse{Type: api.DownloadResponse_PRESIGNED_URL, Status: "Success", Url: []string{""}}, nil
}

func logManifestResult(id string, res shared.ManifestResult, err error) {
	if err != nil {
		log.Errorf("Download of %s stopped: %v", id, err)
	}
	log.Infof("Download of %s: %d files downloaded, %d failed, %d skipped, %d blocked by the malware scan",
		id, res.Downloaded, res.Failed, res.Skipped, res.Blocked)
}
