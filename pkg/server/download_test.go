package server

import (
	"context"
	"testing"
	"time"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// running starts a download that runs until cancelled, and returns a
// channel closed when it stops.
func running(s *agentServer, id string) <-chan struct{} {
	stopped := make(chan struct{})
	started := make(chan struct{})
	s.startDownload(id, func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(stopped)
	})
	<-started
	return stopped
}

func stops(t *testing.T, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(time.Second):
		t.Fatal("the download is still running")
	}
}

func TestCancelDownload(t *testing.T) {
	s := &agentServer{}
	ds1 := running(s, "N:dataset:1")
	ds2 := running(s, "N:dataset:2")
	pkg := running(s, "N:package:3")

	id := "N:dataset:1"
	resp, err := s.CancelDownload(context.Background(), &api.CancelDownloadRequest{Id: &id})
	require.NoError(t, err)
	assert.Equal(t, "Cancelled 1 download.", resp.Status)
	stops(t, ds1)
	select {
	case <-ds2:
		t.Fatal("another dataset's download was cancelled")
	default:
	}

	resp, err = s.CancelDownload(context.Background(), &api.CancelDownloadRequest{CancelAll: true})
	require.NoError(t, err)
	assert.Equal(t, "Cancelled 2 downloads.", resp.Status)
	stops(t, ds2)
	stops(t, pkg)

	assert.Eventually(t, func() bool {
		resp, err := s.CancelDownload(context.Background(), &api.CancelDownloadRequest{CancelAll: true})
		return err == nil && resp.Status == "No download to cancel."
	}, time.Second, 10*time.Millisecond, "finished downloads are forgotten")

	_, err = s.CancelDownload(context.Background(), &api.CancelDownloadRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCheckFreeSpace(t *testing.T) {
	dir := t.TempDir()
	assert.NoError(t, checkFreeSpace(dir, 1024))
	err := checkFreeSpace(dir+"/new", 1<<62)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), "--force")
}
