package manifest

import (
	"bytes"
	"context"
	v1 "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"io"
	"net"
	"strconv"
	"testing"
)

func setupCommand() (removeCmd *cobra.Command, outBuffer *bytes.Buffer, errBuffer *bytes.Buffer) {
	removeCmd = NewRemoveCmd()
	var out, err bytes.Buffer
	outBuffer, errBuffer = &out, &err
	removeCmd.SetOut(outBuffer)
	removeCmd.SetErr(errBuffer)
	return
}

func readBuffer(t require.TestingT, buffer *bytes.Buffer) string {
	buffBytes, err := io.ReadAll(buffer)
	require.NoError(t, err)
	return string(buffBytes)
}

func TestRemoveCmd_Help(t *testing.T) {
	removeCmd, outBuffer, errBuffer := setupCommand()
	removeCmd.SetArgs([]string{"--help"})
	require.NoError(t, removeCmd.Execute())

	output := readBuffer(t, outBuffer)
	assert.Contains(t, output, "remove -m MANIFEST-ID SOURCE-PATH")

	assert.Empty(t, errBuffer)
}

func TestRemoveCmd_BadArgs(t *testing.T) {
	tests := []struct {
		scenario    string
		args        []string
		expectedErr string
	}{
		{"no args", []string{}, "accepts 1 arg(s), received 0"},
		{"only positional arg", []string{"path/to/file"}, `required flag(s) "manifest_id" not set`},
		{"only flag, no value", []string{"-m"}, "flag needs an argument: 'm' in -m"},
		{"wrong manifest id type", []string{"-m", "my-manifest"}, `invalid argument "my-manifest" for "-m, --manifest_id" flag: strconv.ParseInt: parsing "my-manifest": invalid syntax`},
		{"wrong flag", []string{"-m", "3", "--not-a-flag"}, "unknown flag: --not-a-flag"},
		{"no path", []string{"-m", "3"}, "accepts 1 arg(s), received 0"},
		{"more than one path", []string{"-m", "3", "path/to/file1.txt", "path/to/file2.txt"}, "accepts 1 arg(s), received 2"},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			removeCmd, outBuffer, errBuffer := setupCommand()
			removeCmd.SetArgs(tt.args)
			err := removeCmd.Execute()

			assert.Error(t, err)
			//fmt.Println("err", err)
			assert.Equal(t, tt.expectedErr, err.Error())

			stdout := readBuffer(t, outBuffer)
			// fmt.Println("stdout", stdout)
			assert.Contains(t, stdout, "remove -m MANIFEST-ID SOURCE-PATH")

			stderr := readBuffer(t, errBuffer)
			//fmt.Println("stderr", stderr)
			assert.Contains(t, stderr, tt.expectedErr)

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
			removeCmd, outBuffer, errBuffer := setupCommand()
			removeCmd.SetArgs(tt.args)
			require.NoError(t, removeCmd.Execute())

			assert.Empty(t, errBuffer)

			out := readBuffer(t, outBuffer)
			assert.Contains(t, out, manifestIDString)
			assert.Contains(t, out, path)
			assert.Contains(t, out, expectedStatus)
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

// SetViper sets the given key to the given value in viper. Since viper uses global singletons, a cleanup function is registered
// on t that restores any previous value associated with key.
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
