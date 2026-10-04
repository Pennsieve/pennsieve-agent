package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

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
		r := req.GetDataset()
		return s.downloadWorkspace(ctx, client, workspaceDownload{
			id: r.DatasetId, datasetId: r.DatasetId, request: download.ManifestRequest{NodeIds: r.NodeIds},
			target: r.TargetFolder, force: r.Force, smaller: "download part of the dataset with --node",
			// A whole dataset also gets its workspace manifest, as before.
			workspaceManifest: len(r.NodeIds) == 0,
		})

	case api.DownloadRequest_PUBLIC:
		r := req.GetPublic()
		return s.downloadPublic(ctx, client, publicDownload{
			request: download.PublicManifestRequest{DatasetId: r.DatasetId, Version: int(r.Version), Paths: r.Paths},
			target:  r.TargetFolder, force: r.Force, smaller: "download part of the version with --path",
		})

	case api.DownloadRequest_SELECTION:
		return s.downloadSelection(ctx, client, req.GetSelection())
	}
	return nil, fmt.Errorf("unknown download type: %v", req.Type)
}

// workspaceDownload is a selection of a workspace dataset to download.
type workspaceDownload struct {
	// id names the download for CancelDownload.
	id        string
	datasetId string
	request   download.ManifestRequest
	target    string
	force     bool
	// smaller says how to download less, for the free-space error.
	smaller string
	// workspaceManifest also saves the dataset's manifest in .pennsieve.
	workspaceManifest bool
}

// downloadWorkspace downloads a selection of a dataset into its target
// folder through download-service. The first page of links is requested
// before returning, so a selection that can't be downloaded fails here; the
// files download in the background.
func (s *agentServer) downloadWorkspace(ctx context.Context, client *pennsieve.Client, d workspaceDownload) (*api.DownloadResponse, error) {
	first, err := client.Download.GetManifestPage(ctx, d.datasetId, d.request)
	if err != nil {
		log.Errorf("Download failed: %v", err)
		return nil, downloadError(err)
	}

	if !d.force {
		if err := checkFreeSpace(d.target, first.Header.Size, d.smaller); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(d.target, os.ModePerm); err != nil {
		log.Errorf("Failed to create target path: %v", err)
		return nil, err
	}
	downloader := shared.NewDownloader(s, client)

	// The files don't need the workspace manifest, so a failure doesn't stop
	// the download.
	if d.workspaceManifest {
		if err := downloadWorkspaceManifest(ctx, client, &downloader, d.datasetId, d.target); err != nil {
			log.Warnf("Downloading without .pennsieve/manifest.json: %v", err)
		}
	}

	log.Infof("Downloading %d files (%d bytes) of %s to %s",
		first.Header.Count, first.Header.Size, d.datasetId, d.target)
	s.startDownload(d.id, func(ctx context.Context) {
		res, err := downloader.DownloadManifest(ctx, shared.ManifestDownload{
			DatasetId: d.datasetId,
			Request:   d.request,
			Target: func(f download.ManifestFile) string {
				return shared.SafeJoin(d.target, append(f.Path, f.FileName)...)
			},
		}, first)
		logManifestResult(d.id, res, err)
	})

	return &api.DownloadResponse{
		Type: api.DownloadResponse_DOWNLOAD, Status: "Success", Url: []string{""},
		FileCount: int64(first.Header.Count), TotalBytes: first.Header.Size,
	}, nil
}

// publicDownload is a selection of a published (Discover) dataset to
// download.
type publicDownload struct {
	// id names the download for CancelDownload; empty is the dataset id.
	id      string
	request download.PublicManifestRequest
	target  string
	force   bool
	smaller string
}

// downloadPublic downloads a published selection into its target folder,
// as downloadWorkspace does. The user's daily allowance for published data
// is spent by the first page, before returning.
func (s *agentServer) downloadPublic(ctx context.Context, client *pennsieve.Client, d publicDownload) (*api.DownloadResponse, error) {
	first, err := client.Download.GetPublicManifestPage(ctx, d.request)
	if err != nil {
		log.Errorf("Download failed: %v", err)
		return nil, downloadError(err)
	}

	if !d.force {
		if err := checkFreeSpace(d.target, first.Header.Size, d.smaller); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(d.target, os.ModePerm); err != nil {
		log.Errorf("Failed to create target path: %v", err)
		return nil, err
	}

	// The rest of the pages come from the version the first page resolved,
	// even if a newer one is published meanwhile.
	req := d.request
	if req.SelectionId == "" {
		req.DatasetId, req.Version = first.Header.DatasetId, first.Header.Version
	}
	id := d.id
	if id == "" {
		id = strconv.FormatInt(first.Header.DatasetId, 10)
	}

	downloader := shared.NewDownloader(s, client)
	log.Infof("Downloading %d files (%d bytes) of published dataset %d version %d to %s",
		first.Header.Count, first.Header.Size, first.Header.DatasetId, first.Header.Version, d.target)
	s.startDownload(id, func(ctx context.Context) {
		res, err := downloader.DownloadPublicManifest(ctx, shared.PublicManifestDownload{
			Request: req,
			Target: func(f download.PublicManifestFile) string {
				return shared.SafeJoin(d.target, append(f.Path, f.FileName)...)
			},
		}, first)
		logManifestResult(id, res, err)
	})

	return &api.DownloadResponse{
		Type: api.DownloadResponse_DOWNLOAD, Status: "Success", Url: []string{""},
		FileCount: int64(first.Header.Count), TotalBytes: first.Header.Size,
		PublicDatasetId: first.Header.DatasetId, PublicVersion: int32(first.Header.Version),
	}, nil
}

// downloadSelection downloads a selection saved in the app or on Discover.
// The selection only says where to download from; download-service resolves
// its files with the user's own access.
func (s *agentServer) downloadSelection(ctx context.Context, client *pennsieve.Client, r *api.DownloadSelectionRequest) (*api.DownloadResponse, error) {
	if r.GetSelectionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "name the selection to download")
	}
	sel, err := client.Download.GetSelection(ctx, r.SelectionId)
	if err != nil {
		log.Errorf("Download failed: %v", err)
		return nil, downloadError(err)
	}
	smaller := "select fewer files and download the new selection"
	switch sel.Kind {
	case download.SelectionWorkspace:
		return s.downloadWorkspace(ctx, client, workspaceDownload{
			id: sel.Id, datasetId: sel.DatasetNodeId, request: download.ManifestRequest{SelectionId: sel.Id},
			target: r.TargetFolder, force: r.Force, smaller: smaller,
		})
	case download.SelectionPublic:
		return s.downloadPublic(ctx, client, publicDownload{
			id: sel.Id, request: download.PublicManifestRequest{SelectionId: sel.Id},
			target: r.TargetFolder, force: r.Force, smaller: smaller,
		})
	}
	return nil, status.Errorf(codes.Unimplemented, "this agent can't download a selection of kind %q; update the agent", sel.Kind)
}

// downloadError carries download-service's refusal to the command line: its
// message, with a status code the CLI understands.
func downloadError(err error) error {
	var httpErr *pennsieve.HTTPError
	if !errors.As(err, &httpErr) {
		return err
	}
	code := codes.Unknown
	switch httpErr.StatusCode {
	case http.StatusBadRequest:
		code = codes.InvalidArgument
	case http.StatusUnauthorized:
		code = codes.Unauthenticated
	case http.StatusForbidden:
		code = codes.PermissionDenied
	case http.StatusNotFound, http.StatusGone:
		code = codes.NotFound
	case http.StatusConflict, http.StatusRequestEntityTooLarge:
		code = codes.FailedPrecondition
	case http.StatusTooManyRequests:
		code = codes.ResourceExhausted
	case http.StatusServiceUnavailable:
		code = codes.Unavailable
	}
	msg := httpErr.Message
	if msg == "" {
		msg = httpErr.Error()
	}
	return status.Error(code, msg)
}

// checkFreeSpace refuses a download larger than the free space on the
// target folder's disk. smaller says how to download less.
func checkFreeSpace(targetFolder string, size int64, smaller string) error {
	free, err := shared.FreeBytes(targetFolder)
	if err != nil {
		log.Warnf("Cannot check the free space for %s: %v", targetFolder, err)
		return nil
	}
	if size <= 0 || uint64(size) <= free {
		return nil
	}
	return status.Errorf(codes.FailedPrecondition,
		"this download is %s but the disk of %s has %s free: choose another folder, %s, or use --force to download anyway",
		shared.HumanBytes(size), targetFolder, shared.HumanBytes(int64(free)), smaller)
}

func downloadWorkspaceManifest(ctx context.Context, client *pennsieve.Client, downloader shared.Downloader, datasetId, target string) error {
	manifestResponse, err := client.Dataset.GetManifest(ctx, datasetId)
	if err != nil {
		log.Errorf("Download failed: %v", err)
		return err
	}
	if err := os.MkdirAll(filepath.Join(target, ".pennsieve"), os.ModePerm); err != nil {
		log.Errorf("Failed to create target path: %v", err)
		return err
	}
	manifestLocation := filepath.Join(target, ".pennsieve", "manifest.json")
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
