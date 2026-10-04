package shared

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/download"
	log "github.com/sirupsen/logrus"
)

// MaxManifestNodeIds is the most nodeIds download-service takes in one
// manifest request.
const MaxManifestNodeIds = 1000

// ManifestDownload is a selection of a dataset to download through
// download-service.
type ManifestDownload struct {
	DatasetId string
	Request   download.ManifestRequest
	// Target is where a file is saved; "" skips it.
	Target func(f download.ManifestFile) string
	// Done, if set, is called after each file is saved, from the download
	// workers: it must be safe for concurrent use.
	Done func(f download.ManifestFile, target string)
	// Workers download a page's files in parallel (default 5).
	Workers int
}

// ManifestResult counts what a manifest download did.
type ManifestResult struct {
	Downloaded int
	Failed     int
	Skipped    int
	// Blocked files didn't pass the malware scan and aren't served.
	Blocked int
}

// DownloadManifest downloads every file of a selection. It asks
// download-service for one page of signed links at a time and downloads the
// page before asking for the next, so links are used soon after they're
// signed; one that expires anyway is signed again. first is the selection's
// first page when the caller already has it, else nil.
func (s *downloader) DownloadManifest(ctx context.Context, d ManifestDownload, first *download.ManifestPage) (ManifestResult, error) {
	var res ManifestResult
	handle := func(page *download.ManifestPage) error {
		for _, b := range page.Blocked {
			log.Warnf("Not downloading %s: it didn't pass the malware scan (%s)", b.FileName, b.ScanStatus)
		}
		res.Blocked += len(page.Blocked)
		s.downloadPage(ctx, d, page.Data, &res)
		return ctx.Err()
	}

	req := d.Request
	if first != nil {
		if err := handle(first); err != nil || first.Next == "" {
			return res, err
		}
		req.Cursor = first.Next
	}
	err := s.pennsieveClient.Download.WalkManifest(ctx, d.DatasetId, req, handle)
	return res, err
}

func (s *downloader) downloadPage(ctx context.Context, d ManifestDownload, files []download.ManifestFile, res *ManifestResult) {
	var mu sync.Mutex
	inParallel(ctx, d.Workers, files, func(f download.ManifestFile) {
		target := d.Target(f)
		if target == "" {
			mu.Lock()
			res.Skipped++
			mu.Unlock()
			return
		}
		err := downloadTo(target, func() error { return s.downloadOrResign(ctx, d.DatasetId, f, target) })
		if err == nil && d.Done != nil {
			d.Done(f, target)
		}
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			log.Errorf("Download of %s failed: %v", target, err)
			res.Failed++
		} else {
			res.Downloaded++
		}
	})
}

// inParallel runs fn on items with workers goroutines (default 5), and
// stops handing out items once ctx is done.
func inParallel[T any](ctx context.Context, workers int, items []T, fn func(T)) {
	if workers <= 0 {
		workers = 5
	}
	jobs := make(chan T)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				fn(item)
			}
		}()
	}
	for _, item := range items {
		if ctx.Err() != nil {
			break
		}
		jobs <- item
	}
	close(jobs)
	wg.Wait()
}

// downloadTo makes target's folder and runs fetch. A file fetch created and
// couldn't finish is removed, so a cancelled download leaves only whole
// files.
func downloadTo(target string, fetch func() error) error {
	if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
		return err
	}
	_, statErr := os.Stat(target)
	err := fetch()
	if err != nil && os.IsNotExist(statErr) {
		os.Remove(target)
	}
	return err
}

func (s *downloader) downloadOrResign(ctx context.Context, datasetId string, f download.ManifestFile, target string) error {
	_, err := s.DownloadFileFromPresignedUrl(ctx, f.URL, target, f.NodeId)
	var status *StatusError
	if !errors.As(err, &status) || status.StatusCode != http.StatusForbidden {
		return err
	}

	// The link expired before its turn: sign this file again. A package is
	// one file, so its node id is enough.
	page, err := s.pennsieveClient.Download.GetManifestPage(ctx, datasetId, download.ManifestRequest{
		NodeIds: []string{f.NodeId},
	})
	if err != nil {
		return err
	}
	for _, again := range page.Data {
		if again.NodeId == f.NodeId {
			_, err = s.DownloadFileFromPresignedUrl(ctx, again.URL, target, f.NodeId)
			return err
		}
	}
	return fmt.Errorf("%s is no longer in the dataset", f.FileName)
}

// SafeJoin joins names from the platform onto root, keeping the result
// inside root: a name can't climb out with "..", or add folders with a
// separator.
func SafeJoin(root string, names ...string) string {
	parts := []string{root}
	for _, n := range names {
		n = strings.NewReplacer("/", "_", "\\", "_").Replace(n)
		if n == "" || n == "." || n == ".." {
			n = "_"
		}
		parts = append(parts, n)
	}
	return filepath.Join(parts...)
}
