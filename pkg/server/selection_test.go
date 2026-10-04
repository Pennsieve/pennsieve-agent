package server

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

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/download"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	workspaceSel = "sel_aaaaaaaaaaaaaaaaaaaaaaaaaa"
	publicSel    = "sel_bbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// selectionService fakes download-service: one workspace and one public
// selection, each resolving to one file. It records the manifest requests.
func selectionService(t *testing.T) (*pennsieve.Client, *[]string) {
	var mu sync.Mutex
	var calls []string
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/downloads/selections/", func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Base(r.URL.Path) {
		case workspaceSel:
			fmt.Fprintf(w, `{"id":%q,"kind":"workspace","datasetNodeId":"N:dataset:1","expiresAt":"2026-10-06T12:00:00.000Z"}`, workspaceSel)
		case publicSel:
			fmt.Fprintf(w, `{"id":%q,"kind":"public","datasetId":5347,"version":2,"expiresAt":"2026-10-06T12:00:00.000Z"}`, publicSel)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"selection not found, or expired; select the files again"}`)
		}
	})
	mux.HandleFunc("/downloads/manifests", func(w http.ResponseWriter, r *http.Request) {
		var req download.ManifestRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		calls = append(calls, fmt.Sprintf("workspace %s %s", r.URL.Query().Get("dataset_id"), req.SelectionId))
		mu.Unlock()
		require.NoError(t, json.NewEncoder(w).Encode(download.ManifestPage{
			Header: download.ManifestHeader{Count: 1, Size: 4},
			Data: []download.ManifestFile{{
				NodeId: "N:package:1", FileId: 1, FileName: "a.csv", Path: []string{"study"}, URL: srv.URL + "/s3/a", Size: 4,
			}},
		}))
	})
	mux.HandleFunc("/downloads/public/manifests", func(w http.ResponseWriter, r *http.Request) {
		var req download.PublicManifestRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		calls = append(calls, fmt.Sprintf("public %d %d %s", req.DatasetId, req.Version, req.SelectionId))
		mu.Unlock()
		require.NoError(t, json.NewEncoder(w).Encode(download.PublicManifestPage{
			Header: download.PublicManifestHeader{DatasetId: 5347, Version: 2, Count: 1, Size: 4},
			Data:   []download.PublicManifestFile{{FileName: "p.csv", Path: []string{"files"}, URL: srv.URL + "/s3/p", Size: 4}},
		}))
	})
	mux.HandleFunc("/s3/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "data") })

	client := pennsieve.NewClient(pennsieve.APIParams{ApiHost2: srv.URL})
	client.APISession = pennsieve.APISession{Token: "t", Expiration: time.Now().Add(time.Hour)}
	return client, &calls
}

func downloaded(t *testing.T, path string) {
	t.Helper()
	assert.Eventually(t, func() bool {
		b, err := os.ReadFile(path)
		return err == nil && string(b) == "data"
	}, 2*time.Second, 10*time.Millisecond, path)
}

func TestDownloadWorkspaceSelection(t *testing.T) {
	client, calls := selectionService(t)
	s := &agentServer{}
	root := t.TempDir()

	resp, err := s.downloadSelection(context.Background(), client, &api.DownloadSelectionRequest{
		SelectionId: workspaceSel, TargetFolder: root,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.FileCount)
	assert.Zero(t, resp.PublicDatasetId)
	downloaded(t, filepath.Join(root, "study", "a.csv"))
	assert.Equal(t, []string{"workspace N:dataset:1 " + workspaceSel}, *calls,
		"the selection's dataset, and the selection rather than its ids")
	assert.NoDirExists(t, filepath.Join(root, ".pennsieve"), "a selection isn't the whole dataset")
}

func TestDownloadPublicSelection(t *testing.T) {
	client, calls := selectionService(t)
	s := &agentServer{}
	root := t.TempDir()

	resp, err := s.downloadSelection(context.Background(), client, &api.DownloadSelectionRequest{
		SelectionId: publicSel, TargetFolder: root,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5347), resp.PublicDatasetId)
	assert.Equal(t, int32(2), resp.PublicVersion)
	downloaded(t, filepath.Join(root, "files", "p.csv"))
	assert.Equal(t, []string{"public 0 0 " + publicSel}, *calls, "only the selection: it fixes the version")
}

func TestDownloadPublicPinsTheResolvedVersion(t *testing.T) {
	client, _ := selectionService(t)
	s := &agentServer{}
	root := t.TempDir()

	resp, err := s.downloadPublic(context.Background(), client, publicDownload{
		request: download.PublicManifestRequest{DatasetId: 5347, Paths: []string{"files"}}, target: root,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), resp.PublicVersion, "latest resolved to a number")
	downloaded(t, filepath.Join(root, "files", "p.csv"))
}

func TestDownloadUnknownSelection(t *testing.T) {
	client, calls := selectionService(t)
	s := &agentServer{}

	_, err := s.downloadSelection(context.Background(), client, &api.DownloadSelectionRequest{
		SelectionId: "sel_cccccccccccccccccccccccccc", TargetFolder: t.TempDir(),
	})
	st := status.Convert(err)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "selection not found, or expired; select the files again", st.Message(), "the service's message, as is")
	assert.Empty(t, *calls)

	_, err = s.downloadSelection(context.Background(), client, &api.DownloadSelectionRequest{TargetFolder: t.TempDir()})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestDownloadError(t *testing.T) {
	for httpStatus, code := range map[int]codes.Code{
		http.StatusUnauthorized:          codes.Unauthenticated,
		http.StatusForbidden:             codes.PermissionDenied,
		http.StatusNotFound:              codes.NotFound,
		http.StatusGone:                  codes.NotFound,
		http.StatusRequestEntityTooLarge: codes.FailedPrecondition,
		http.StatusTooManyRequests:       codes.ResourceExhausted,
	} {
		err := downloadError(&pennsieve.HTTPError{StatusCode: httpStatus, Message: "why"})
		assert.Equal(t, code, status.Code(err), httpStatus)
		assert.Equal(t, "why", status.Convert(err).Message())
	}
	plain := fmt.Errorf("connection refused")
	assert.Equal(t, plain, downloadError(plain))
}
