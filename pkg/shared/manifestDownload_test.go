package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/download"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noSubscribers struct{}

func (noSubscribers) GetSubscribers() sync.Map { return sync.Map{} }

// manifestServer serves a two-page manifest of three files and one blocked
// file. The first link to f2 has expired; signing it again works.
func manifestServer(t *testing.T) (*httptest.Server, *[]download.ManifestRequest) {
	var mu sync.Mutex
	var requests []download.ManifestRequest
	expired := true
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	file := func(id int64, name string, path []string, sig string) download.ManifestFile {
		return download.ManifestFile{
			NodeId: fmt.Sprintf("N:package:%d", id), FileId: id, FileName: name, Path: path,
			URL: fmt.Sprintf("%s/s3/%d?%s", srv.URL, id, sig), Size: 4,
		}
	}
	mux.HandleFunc("/downloads/manifests", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "N:dataset:1", r.URL.Query().Get("dataset_id"))
		var req download.ManifestRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		requests = append(requests, req)
		mu.Unlock()

		page := download.ManifestPage{Header: download.ManifestHeader{Count: 3, Size: 12, BlockedCount: 1}}
		switch {
		case len(req.FileIds) == 1 && req.FileIds[0] == 2:
			page.Data = []download.ManifestFile{file(2, "b.csv", []string{"study"}, "fresh")}
		case req.Cursor == "":
			page.Data = []download.ManifestFile{file(1, "a.csv", []string{"study"}, "sig"), file(2, "b.csv", []string{"study"}, "old")}
			page.Blocked = []download.BlockedFile{{NodeId: "N:package:9", FileName: "virus.exe", ScanStatus: "infected"}}
			page.Next = "c1"
		case req.Cursor == "c1":
			page.Data = []download.ManifestFile{file(3, "c.csv", []string{"study", "sub"}, "sig")}
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

func testDownloader(srv *httptest.Server) downloader {
	client := pennsieve.NewClient(pennsieve.APIParams{ApiHost2: srv.URL})
	client.APISession = pennsieve.APISession{Token: "t", Expiration: time.Now().Add(time.Hour)}
	return NewDownloader(noSubscribers{}, client)
}

func TestDownloadManifest(t *testing.T) {
	srv, requests := manifestServer(t)
	defer srv.Close()
	d := testDownloader(srv)
	root := t.TempDir()

	var mu sync.Mutex
	var done []string
	res, err := d.DownloadManifest(context.Background(), ManifestDownload{
		DatasetId: "N:dataset:1",
		Target: func(f download.ManifestFile) string {
			return SafeJoin(root, append(f.Path, f.FileName)...)
		},
		Done: func(f download.ManifestFile, target string) {
			mu.Lock()
			defer mu.Unlock()
			done = append(done, f.FileName)
		},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, ManifestResult{Downloaded: 3, Blocked: 1}, res)
	assert.ElementsMatch(t, []string{"a.csv", "b.csv", "c.csv"}, done)

	for _, p := range []string{"study/a.csv", "study/b.csv", "study/sub/c.csv"} {
		b, err := os.ReadFile(filepath.Join(root, p))
		if assert.NoError(t, err, p) {
			assert.Equal(t, "data", string(b))
		}
	}
	assert.NoFileExists(t, filepath.Join(root, "virus.exe"))

	require.Len(t, *requests, 3, "two pages, and the expired link signed again")
	var cursors []string
	for _, r := range *requests {
		if len(r.FileIds) > 0 {
			assert.Equal(t, download.ManifestRequest{NodeIds: []string{"N:package:2"}, FileIds: []int64{2}}, r)
			continue
		}
		cursors = append(cursors, r.Cursor)
	}
	assert.Equal(t, []string{"", "c1"}, cursors)
}

func TestDownloadManifestContinuesFromAFirstPage(t *testing.T) {
	srv, requests := manifestServer(t)
	defer srv.Close()
	d := testDownloader(srv)
	root := t.TempDir()

	first := &download.ManifestPage{Next: "c1"}
	res, err := d.DownloadManifest(context.Background(), ManifestDownload{
		DatasetId: "N:dataset:1",
		Target: func(f download.ManifestFile) string {
			if f.FileId == 3 {
				return ""
			}
			return SafeJoin(root, f.FileName)
		},
	}, first)
	require.NoError(t, err)
	assert.Equal(t, ManifestResult{Skipped: 1}, res)
	require.Len(t, *requests, 1)
	assert.Equal(t, "c1", (*requests)[0].Cursor, "the first page isn't requested again")
}

func TestSafeJoin(t *testing.T) {
	root := filepath.FromSlash("/data")
	assert.Equal(t, filepath.FromSlash("/data/study/a.csv"), SafeJoin(root, "study", "a.csv"))
	assert.Equal(t, filepath.FromSlash("/data/_/_/etc_passwd"), SafeJoin(root, "..", "", "etc/passwd"))
	assert.Equal(t, filepath.FromSlash("/data/a_b"), SafeJoin(root, `a\b`))
}
