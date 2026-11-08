package cloud

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "JWT"), srv
}

func TestClient_Register(t *testing.T) {
	c, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/agents/register", r.URL.Path)
		assert.Equal(t, "Bearer JWT", r.Header.Get("Authorization"))
		var req RegisterRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "inst-1", req.InstallationId)
		_ = json.NewEncoder(w).Encode(RegisterResponse{
			AgentId: "a1", AgentSecret: "s1", WSURL: "wss://x", HeartbeatSec: 270,
		})
	})

	resp, err := c.Register(context.Background(), RegisterRequest{InstallationId: "inst-1"})
	require.NoError(t, err)
	assert.Equal(t, "a1", resp.AgentId)
	assert.Equal(t, "s1", resp.AgentSecret)
}

func TestClient_Register_RequiresJWT(t *testing.T) {
	c := NewClient("http://example", "")
	_, err := c.Register(context.Background(), RegisterRequest{InstallationId: "x"})
	require.Error(t, err)
}

func TestClient_WSToken_NoJWT(t *testing.T) {
	// /agents/ws-token does NOT require the user JWT.
	c, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"), "ws-token must not require user auth")
		_ = json.NewEncoder(w).Encode(WSTokenResponse{Token: "tok", ExpiresAt: 1})
	})
	c.JWT = "" // even without JWT this works
	resp, err := c.WSToken(context.Background(), WSTokenRequest{AgentId: "a", AgentSecret: "s"})
	require.NoError(t, err)
	assert.Equal(t, "tok", resp.Token)
}

func TestClient_Reconcile(t *testing.T) {
	c, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/agents/a1/reconcile", r.URL.Path)
		var req ReconcileRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "a1", req.AgentId)
		_ = json.NewEncoder(w).Encode(ReconcileResponse{
			Outstanding: []OutstandingCommand{
				{CommandId: "c1", Type: "ping", Payload: json.RawMessage(`{}`)},
			},
		})
	})
	resp, err := c.Reconcile(context.Background(), "a1", ReconcileRequest{AgentId: "a1"})
	require.NoError(t, err)
	require.Len(t, resp.Outstanding, 1)
	assert.Equal(t, "c1", resp.Outstanding[0].CommandId)
}

func TestClient_NonOK(t *testing.T) {
	c, _ := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, "denied")
	})
	_, err := c.Register(context.Background(), RegisterRequest{InstallationId: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}
