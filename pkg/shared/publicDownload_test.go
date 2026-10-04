package shared

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/download"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func b64sum(s string) string {
	sum := sha256.Sum256([]byte(s))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// publicServer serves a two-page published selection (dataset 5347, version
// 2): a.csv with a matching checksum, b.csv whose first link has expired,
// c.csv whose checksum doesn't match, and one skipped file.
func publicServer(t *testing.T) (*httptest.Server, *[]download.PublicManifestRequest) {
	var mu sync.Mutex
	var requests []download.PublicManifestRequest
	expired := true
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	file := func(name string, path []string, sig, sum string) download.PublicManifestFile {
		return download.PublicManifestFile{
			FileName: name, Path: path, URL: fmt.Sprintf("%s/s3/%s?%s", srv.URL, name, sig), Size: 4, SHA256: sum,
		}
	}
	mux.HandleFunc("/downloads/public/manifests", func(w http.ResponseWriter, r *http.Request) {
		var req download.PublicManifestRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		requests = append(requests, req)
		mu.Unlock()

		page := download.PublicManifestPage{Header: download.PublicManifestHeader{DatasetId: 5347, Version: 2, Count: 3, Size: 12}}
		switch {
		case len(req.Paths) == 1 && req.Paths[0] == "study/b.csv":
			page.Data = []download.PublicManifestFile{file("b.csv", []string{"study"}, "fresh", "")}
		case req.Cursor == "":
			page.Data = []download.PublicManifestFile{
				file("a.csv", []string{"study"}, "sig", b64sum("data")),
				file("b.csv", []string{"study"}, "old", ""),
			}
			page.Skipped = []download.SkippedFile{{FileName: "legacy.bin", Path: []string{"study"}, Reason: "no_object_version"}}
			page.Next = "p1"
		case req.Cursor == "p1":
			page.Data = []download.PublicManifestFile{file("c.csv", []string{"study", "sub"}, "sig", b64sum("other"))}
		}
		require.NoError(t, json.NewEncoder(w).Encode(page))
	})
	mux.HandleFunc("/s3/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.RawQuery == "old" && expired {
			expired = false
			w.WriteHeader(http.StatusForbidden)
			return
		}
		fmt.Fprint(w, "data")
	})
	return srv, &requests
}

func TestDownloadPublicManifest(t *testing.T) {
	srv, requests := publicServer(t)
	defer srv.Close()
	d := testDownloader(srv)
	root := t.TempDir()

	res, err := d.DownloadPublicManifest(context.Background(), PublicManifestDownload{
		Request: download.PublicManifestRequest{DatasetId: 5347},
		Target: func(f download.PublicManifestFile) string {
			return SafeJoin(root, append(f.Path, f.FileName)...)
		},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, ManifestResult{Downloaded: 2, Failed: 1, Skipped: 1}, res)

	for _, p := range []string{"study/a.csv", "study/b.csv"} {
		b, err := os.ReadFile(filepath.Join(root, p))
		if assert.NoError(t, err, p) {
			assert.Equal(t, "data", string(b))
		}
	}
	assert.NoFileExists(t, filepath.Join(root, "study/sub/c.csv"), "a file that doesn't match its checksum is removed")

	require.Len(t, *requests, 3, "two pages, and the expired link signed again")
	var cursors []string
	for _, r := range *requests {
		if len(r.Paths) > 0 {
			assert.Equal(t, download.PublicManifestRequest{DatasetId: 5347, Version: 2, Paths: []string{"study/b.csv"}}, r,
				"signed again for the same version and path")
			continue
		}
		cursors = append(cursors, r.Cursor)
	}
	assert.Equal(t, []string{"", "p1"}, cursors)
}

func TestVerifySHA256(t *testing.T) {
	dir := t.TempDir()
	write := func() string {
		p := filepath.Join(dir, "f.csv")
		require.NoError(t, os.WriteFile(p, []byte("data"), 0o644))
		return p
	}
	sum := sha256.Sum256([]byte("data"))

	assert.NoError(t, verifySHA256(write(), b64sum("data")), "base64, as Discover records it")
	assert.NoError(t, verifySHA256(write(), hex.EncodeToString(sum[:])), "hex")
	assert.NoError(t, verifySHA256(write(), ""), "no checksum")
	assert.NoError(t, verifySHA256(write(), "abcd-3"), "a multipart checksum isn't a hash of the file")
	assert.FileExists(t, filepath.Join(dir, "f.csv"))

	assert.ErrorIs(t, verifySHA256(write(), b64sum("other")), ErrChecksum)
	assert.NoFileExists(t, filepath.Join(dir, "f.csv"))
}
