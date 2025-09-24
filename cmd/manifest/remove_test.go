package manifest

import (
	"context"
	v1 "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"testing"
)

func TestRemoveCmd_Help(t *testing.T) {
	// Must do everything through parent command since running Execute()
	// on child command runs it from the parent anyway.
	removeCmd := NewRemoveCmd()
	removeCmd.SetArgs([]string{"--help"})
	require.NoError(t, removeCmd.Execute())
}

func TestRemoveCmd_BadArgs(t *testing.T) {
	tests := []struct {
		scenario string
		args     []string
	}{
		{"no args", []string{}},
		{"only positional arg", []string{"path/to/file"}},
		{"only flag, no value", []string{"-m"}},
		{"wrong manifest id type", []string{"-m", "my-manifest"}},
		{"wrong flag", []string{"-m", "3", "--not-a-flag"}},
		{"no path", []string{"-m", "3"}},
		{"more than one path", []string{"-m", "3", "path/to/file1.txt", "path/to/file2.txt"}},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			removeCmd := NewRemoveCmd()
			removeCmd.SetArgs(tt.args)
			err := removeCmd.Execute()
			assert.Error(t, err)
		})
	}
}

func TestRemoveCmd(t *testing.T) {
	manifestID := int32(3)
	path := "path/1/test.txt"
	expectedStatus := "all good"
	port := StartMockGRPC(t,
		newMockRemoveFromManifest(func(_ context.Context, request *v1.RemoveFromManifestRequest) (*v1.SimpleStatusResponse, error) {
			assert.Equal(t, manifestID, request.GetManifestId())
			assert.Equal(t, path, request.GetRemovePath())
			return &v1.SimpleStatusResponse{Status: expectedStatus}, nil
		}))

	SetViper(t, "agent.port", port)

	manifestIDString := strconv.FormatInt(int64(manifestID), 10)

	tests := []struct {
		scenario string
		args     []string
	}{
		{"short form", []string{"-m", manifestIDString, path}},
		{"long form", []string{"--manifest_id", manifestIDString, path}},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			removeCmd := NewRemoveCmd()
			removeCmd.SetArgs(tt.args)
			require.NoError(t, removeCmd.Execute())
		})
	}

}

type removeFromManifestFunc func(ctx context.Context, request *v1.RemoveFromManifestRequest) (*v1.SimpleStatusResponse, error)

type mockRemoveFromManifest struct {
	v1.UnimplementedAgentServer
	f removeFromManifestFunc
}

func newMockRemoveFromManifest(f removeFromManifestFunc) mockRemoveFromManifest {
	return mockRemoveFromManifest{f: f}
}

func (m mockRemoveFromManifest) RemoveFromManifest(ctx context.Context, req *v1.RemoveFromManifestRequest) (*v1.SimpleStatusResponse, error) {
	if m.f == nil {
		panic("mock RemoveFromManifest function not set")
	}
	return m.f(ctx, req)
}

func SetViper(t *testing.T, key string, value any) {
	old := viper.Get(key)
	t.Cleanup(func() {
		viper.Set(key, old)
	})
	viper.Set(key, value)
}

// StartMockGRPC starts a GPRC server that registers mock as the v1.AgentServer
// Server and listener Close() are registered to t's Cleanup() method.
func StartMockGRPC(t *testing.T, mock v1.AgentServer) (port string) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		lis.Close()
	})

	_, port, err = net.SplitHostPort(lis.Addr().String())
	require.NoError(t, err)

	s := grpc.NewServer()
	t.Cleanup(func() {
		s.Stop()
	})
	s.RegisterService(&v1.Agent_ServiceDesc, mock)
	go func() {
		err := s.Serve(lis)
		assert.NoError(t, err)
	}()

	return
}
