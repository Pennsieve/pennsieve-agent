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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	if !requestData.Force {
		if err := checkFreeSpace(requestData.TargetFolder, first.Header.Size); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(requestData.TargetFolder, os.ModePerm); err != nil {
		log.Errorf("Failed to create target path: %v", err)
		return nil, err
	}
	downloader := shared.NewDownloader(s, client)

	// A whole dataset also gets its workspace manifest in the hidden
	// .pennsieve folder, as before. The files don't need it, so a failure
	// doesn't stop the download.
	if len(requestData.NodeIds) == 0 {
		if err := downloadWorkspaceManifest(ctx, client, &downloader, requestData); err != nil {
			log.Warnf("Downloading without .pennsieve/manifest.json: %v", err)
		}
	}

	log.Infof("Downloading %d files (%d bytes) of %s to %s",
		first.Header.Count, first.Header.Size, requestData.DatasetId, requestData.TargetFolder)
	s.startDownload(requestData.DatasetId, func(ctx context.Context) {
		res, err := downloader.DownloadManifest(ctx, shared.ManifestDownload{
			DatasetId: requestData.DatasetId,
			Request:   manifestReq,
			Target: func(f download.ManifestFile) string {
				return shared.SafeJoin(requestData.TargetFolder, append(f.Path, f.FileName)...)
			},
		}, first)
		logManifestResult(requestData.DatasetId, res, err)
	})

	return &api.DownloadResponse{
		Type: api.DownloadResponse_DOWNLOAD, Status: "Success", Url: []string{""},
		FileCount: int64(first.Header.Count), TotalBytes: first.Header.Size,
	}, nil
}

// checkFreeSpace refuses a download larger than the free space on the
// target folder's disk.
func checkFreeSpace(targetFolder string, size int64) error {
	free, err := shared.FreeBytes(targetFolder)
	if err != nil {
		log.Warnf("Cannot check the free space for %s: %v", targetFolder, err)
		return nil
	}
	if size <= 0 || uint64(size) <= free {
		return nil
	}
	return status.Errorf(codes.FailedPrecondition,
		"this download is %s but the disk of %s has %s free: choose another folder, download part of the dataset with --node, or use --force to download anyway",
		shared.HumanBytes(size), targetFolder, shared.HumanBytes(int64(free)))
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
	s.startDownload(requestData.PackageId, func(ctx context.Context) {
		res, err := downloader.DownloadManifest(ctx, shared.ManifestDownload{
			DatasetId: requestData.DatasetId,
			Request:   manifestReq,
			// A package of several files gets a folder of its name (Path).
			Target: func(f download.ManifestFile) string {
				return shared.SafeJoin(".", append(f.Path, f.FileName)...)
			},
		}, first)
		logManifestResult(requestData.PackageId, res, err)
	})

	return &api.DownloadResponse{
		Type: api.DownloadResponse_DOWNLOAD, Status: "Success", Url: []string{""},
		FileCount: int64(first.Header.Count), TotalBytes: first.Header.Size,
	}, nil
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
		s.startDownload(requestData.PackageId, func(ctx context.Context) {
			// Iterate over the files in a package and download serially
			for _, f := range res.Files {
				downloaderImpl := shared.NewDownloader(s, client)
				_, err := downloaderImpl.DownloadFileFromPresignedUrl(ctx, f.URL, f.Name, requestData.PackageId)
				if err != nil {
					log.Errorf("Download failed: %v", err)
				}
			}
		})
	}

	return &api.DownloadResponse{Type: api.DownloadResponse_PRESIGNED_URL, Status: "Success", Url: []string{""}}, nil
}

// downloadKey names a running download: the dataset or package it was
// started for (what CancelDownload takes), and a unique suffix.
type downloadKey struct {
	id  string
	run string
}

// startDownload runs fn in the background until it returns or is cancelled
// by CancelDownload with id, the download's dataset or package.
func (s *agentServer) startDownload(id string, fn func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	key := downloadKey{id: id, run: uuid.NewString()}
	s.downloads.Store(key, cancel)
	go func() {
		defer func() {
			cancel()
			s.downloads.Delete(key)
		}()
		fn(ctx)
	}()
}

// CancelDownload stops the running downloads of a dataset or package, or
// all of them. Files being downloaded stop too, and are removed.
func (s *agentServer) CancelDownload(ctx context.Context, req *api.CancelDownloadRequest) (*api.SimpleStatusResponse, error) {
	if !req.CancelAll && req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "name the dataset or package whose download to cancel, or cancel all")
	}
	cancelled := 0
	s.downloads.Range(func(k, v any) bool {
		if req.CancelAll || k.(downloadKey).id == req.GetId() {
			v.(context.CancelFunc)()
			cancelled++
		}
		return true
	})
	if cancelled == 0 {
		return &api.SimpleStatusResponse{Status: "No download to cancel."}, nil
	}
	log.Infof("Cancelled %d downloads", cancelled)
	return &api.SimpleStatusResponse{Status: fmt.Sprintf("Cancelled %d %s.", cancelled, plural(cancelled, "download", "downloads"))}, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func logManifestResult(id string, res shared.ManifestResult, err error) {
	if err != nil {
		log.Errorf("Download of %s stopped: %v", id, err)
	}
	log.Infof("Download of %s: %d files downloaded, %d failed, %d skipped, %d blocked by the malware scan",
		id, res.Downloaded, res.Failed, res.Skipped, res.Blocked)
}
