package shared

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/download"
	log "github.com/sirupsen/logrus"
)

// PublicManifestDownload is a selection of a published (Discover) dataset to
// download through download-service.
type PublicManifestDownload struct {
	Request download.PublicManifestRequest
	// Target is where a file is saved; "" skips it.
	Target func(f download.PublicManifestFile) string
	// Workers download a page's files in parallel (default 5).
	Workers int
}

// ErrChecksum means a downloaded file doesn't match the checksum recorded
// when it was published.
var ErrChecksum = errors.New("the file doesn't match its published checksum")

// DownloadPublicManifest downloads every file of a published selection, page
// by page like DownloadManifest. Files with a published checksum are
// verified; one that doesn't match is removed and counted as failed. first
// is the selection's first page when the caller already has it, else nil.
func (s *downloader) DownloadPublicManifest(ctx context.Context, d PublicManifestDownload, first *download.PublicManifestPage) (ManifestResult, error) {
	var res ManifestResult
	handle := func(page *download.PublicManifestPage) error {
		for _, f := range page.Skipped {
			log.Warnf("Not downloading %s: it can't be downloaded (%s)", strings.Join(append(slices.Clone(f.Path), f.FileName), "/"), f.Reason)
		}
		res.Skipped += len(page.Skipped)
		s.downloadPublicPage(ctx, d, page, &res)
		return ctx.Err()
	}

	req := d.Request
	if first != nil {
		if err := handle(first); err != nil || first.Next == "" {
			return res, err
		}
		req.Cursor = first.Next
	}
	err := s.pennsieveClient.Download.WalkPublicManifest(ctx, req, handle)
	return res, err
}

func (s *downloader) downloadPublicPage(ctx context.Context, d PublicManifestDownload, page *download.PublicManifestPage, res *ManifestResult) {
	var mu sync.Mutex
	inParallel(ctx, d.Workers, page.Data, func(f download.PublicManifestFile) {
		target := d.Target(f)
		if target == "" {
			mu.Lock()
			res.Skipped++
			mu.Unlock()
			return
		}
		err := downloadTo(target, func() error {
			if err := s.downloadPublicOrResign(ctx, page.Header, f, target); err != nil {
				return err
			}
			return verifySHA256(target, f.SHA256)
		})
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

// downloadPublicOrResign downloads a published file, signing it again if
// its link expired before its turn. The new link is for the same version
// (the header's, also when the request asked for the latest) and path.
func (s *downloader) downloadPublicOrResign(ctx context.Context, header download.PublicManifestHeader, f download.PublicManifestFile, target string) error {
	id := publicFileId(header, f)
	_, err := s.DownloadFileFromPresignedUrl(ctx, f.URL, target, id)
	var status *StatusError
	if !errors.As(err, &status) || status.StatusCode != http.StatusForbidden {
		return err
	}

	path := strings.Join(append(slices.Clone(f.Path), f.FileName), "/")
	page, err := s.pennsieveClient.Download.GetPublicManifestPage(ctx, download.PublicManifestRequest{
		DatasetId: header.DatasetId, Version: header.Version, Paths: []string{path},
	})
	if err != nil {
		return err
	}
	for _, again := range page.Data {
		if again.FileName == f.FileName && slices.Equal(again.Path, f.Path) {
			_, err = s.DownloadFileFromPresignedUrl(ctx, again.URL, target, id)
			return err
		}
	}
	return fmt.Errorf("%s is no longer in version %d", path, header.Version)
}

// publicFileId names a published file's download (for cancelling it):
// dataset, version and path.
func publicFileId(header download.PublicManifestHeader, f download.PublicManifestFile) string {
	return fmt.Sprintf("%d/%d/%s", header.DatasetId, header.Version, strings.Join(append(slices.Clone(f.Path), f.FileName), "/"))
}

// verifySHA256 checks a downloaded file against its published checksum, if
// there is one, and removes it when it doesn't match. Discover records the
// file's SHA-256 in base64, as S3 does; hex is accepted too. Anything else
// (such as a multipart upload's checksum of checksums, "…-N") isn't a hash of
// the whole file and isn't checked.
func verifySHA256(target, published string) error {
	want := sha256Digest(published)
	if want == nil {
		if published != "" {
			log.Debugf("Not checking %s: its published checksum isn't a SHA-256 of the file", target)
		}
		return nil
	}
	file, err := os.Open(target)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(h, file)
	file.Close()
	if err != nil {
		return err
	}
	if !bytes.Equal(h.Sum(nil), want) {
		os.Remove(target)
		return ErrChecksum
	}
	return nil
}

// sha256Digest reads a published SHA-256 in base64 or hex; nil when it's
// neither.
func sha256Digest(published string) []byte {
	if b, err := base64.StdEncoding.DecodeString(published); err == nil && len(b) == sha256.Size {
		return b
	}
	if b, err := hex.DecodeString(published); err == nil && len(b) == sha256.Size {
		return b
	}
	return nil
}
