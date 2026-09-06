package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRegistrar serves /agents/register and /agents/ws-token. It
// records calls so tests can verify what the agent sent.
type fakeRegistrar struct {
	mu       sync.Mutex
	registers []RegisterRequest
	srv      *httptest.Server
}

func newFakeRegistrar(t *testing.T, agentId, secret string) *fakeRegistrar {
	t.Helper()
	r := &fakeRegistrar{}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/agents/register":
			var body RegisterRequest
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			r.mu.Lock()
			r.registers = append(r.registers, body)
			r.mu.Unlock()
			resp := RegisterResponse{
				AgentId: agentId, AgentSecret: secret, WSURL: "wss://x",
				WSTokenURL: r.srv.URL + "/agents/ws-token", HeartbeatSec: 270,
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	t.Cleanup(r.srv.Close)
	return r
}

func TestManager_StartRegistersAndPersists(t *testing.T) {
	store := NewStore(newTestDB(t))
	srv := newFakeRegistrar(t, "agent-1", "secret-1")

	m := NewManager(store, srv.srv.URL, "JWT", "host", "v1.0")
	m.Dialer = &stubDialer{conns: []WSConn{newFakeConn()}}
	m.TokenSource = &stubTokenSource{token: "tok"}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	require.NoError(t, m.Start(ctx, "default"))
	t.Cleanup(m.Stop)

	srv.mu.Lock()
	require.Len(t, srv.registers, 1)
	assert.Equal(t, "default", srv.registers[0].ProfileName)
	assert.NotEmpty(t, srv.registers[0].InstallationId)
	srv.mu.Unlock()

	row, err := store.GetRegistration("default")
	require.NoError(t, err)
	assert.Equal(t, "agent-1", row.AgentId)
	assert.Equal(t, "secret-1", row.AgentSecret)
}

func TestManager_StartReusesExistingRegistration(t *testing.T) {
	store := NewStore(newTestDB(t))

	// Pre-seed the store as if the agent has already registered.
	require.NoError(t, store.UpsertRegistration(Registration{
		ProfileName: "default", AgentId: "old-agent",
		AgentSecret: "old-secret", WSURL: "wss://x",
		WSTokenURL:   "https://x/token", HeartbeatSec: 270,
		RegisteredAt: time.Now(),
	}))

	// Server returns the same agentId and an EMPTY agent secret —
	// indicating "no rotation". Manager must keep the previously
	// stored secret.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body RegisterRequest
		_ = json.NewDecoder(req.Body).Decode(&body)
		assert.Equal(t, "old-agent", body.AgentId)
		_ = json.NewEncoder(w).Encode(RegisterResponse{
			AgentId: "old-agent", AgentSecret: "", WSURL: "wss://x", HeartbeatSec: 270,
		})
	}))
	t.Cleanup(srv.Close)

	m := NewManager(store, srv.URL, "JWT", "host", "v1.0")
	m.Dialer = &stubDialer{conns: []WSConn{newFakeConn()}}
	m.TokenSource = &stubTokenSource{token: "tok"}
	require.NoError(t, m.Start(context.Background(), "default"))
	t.Cleanup(m.Stop)

	row, _ := store.GetRegistration("default")
	assert.Equal(t, "old-agent", row.AgentId)
	assert.Equal(t, "old-secret", row.AgentSecret, "secret preserved when server doesn't rotate")
}

func TestManager_HandleCommand_RunsRegisteredHandler(t *testing.T) {
	store := NewStore(newTestDB(t))

	// We don't need a real REST server for this test — we drive
	// handleCommand directly.
	m := NewManager(store, "http://unused", "JWT", "host", "v1")
	m.reg = Registration{ProfileName: "default", AgentId: "a1", AgentSecret: "s"}
	m.inst = Installation{InstallationId: "inst-1"}

	conn := newFakeConn()
	m.wsCli = NewWSClient(WSClientConfig{
		WSURL:       "wss://x",
		Heartbeat:   time.Hour,
		Dialer:      &stubDialer{conns: []WSConn{conn}},
		TokenSource: &stubTokenSource{token: "tok"},
	})
	require.NoError(t, m.wsCli.Start(context.Background()))
	t.Cleanup(m.wsCli.Stop)

	called := make(chan []byte, 1)
	m.RegisterHandler("ping", func(_ context.Context, payload []byte) ([]byte, error) {
		called <- payload
		return []byte(`{"pong":true}`), nil
	})

	env := Envelope{MessageId: "c1", Kind: KindCommand, Type: "ping", Payload: []byte(`{"x":1}`)}
	go m.handleCommand(context.Background(), env)

	select {
	case p := <-called:
		assert.JSONEq(t, `{"x":1}`, string(p))
	case <-time.After(2 * time.Second):
		t.Fatal("handler not invoked")
	}

	// Wait for terminal status to be persisted.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		seen, _ := store.SeenCommand("c1")
		if seen {
			rows, _ := store.OutstandingCommands()
			if len(rows) == 0 {
				return // c1 is no longer outstanding -> COMPLETED
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("c1 never reached terminal state")
}

func TestManager_HandleCommand_IsIdempotent(t *testing.T) {
	store := NewStore(newTestDB(t))
	m := NewManager(store, "http://unused", "", "host", "v1")
	m.reg = Registration{AgentId: "a1"}

	conn := newFakeConn()
	m.wsCli = NewWSClient(WSClientConfig{
		WSURL: "wss://x", Heartbeat: time.Hour,
		Dialer:      &stubDialer{conns: []WSConn{conn}},
		TokenSource: &stubTokenSource{token: "tok"},
	})
	require.NoError(t, m.wsCli.Start(context.Background()))
	t.Cleanup(m.wsCli.Stop)

	calls := 0
	m.RegisterHandler("ping", func(_ context.Context, _ []byte) ([]byte, error) {
		calls++
		return nil, nil
	})

	env := Envelope{MessageId: "c1", Kind: KindCommand, Type: "ping"}
	m.handleCommand(context.Background(), env)
	m.handleCommand(context.Background(), env)

	// Drain in case work goroutines are still pending.
	time.Sleep(50 * time.Millisecond)
	assert.LessOrEqual(t, calls, 1, "second delivery of same commandId must be a no-op")
}

func TestManager_HandleCommand_NoHandlerReportsFailed(t *testing.T) {
	store := NewStore(newTestDB(t))
	m := NewManager(store, "http://unused", "", "host", "v1")
	m.reg = Registration{AgentId: "a1"}
	conn := newFakeConn()
	m.wsCli = NewWSClient(WSClientConfig{
		WSURL: "wss://x", Heartbeat: time.Hour,
		Dialer:      &stubDialer{conns: []WSConn{conn}},
		TokenSource: &stubTokenSource{token: "tok"},
	})
	require.NoError(t, m.wsCli.Start(context.Background()))
	t.Cleanup(m.wsCli.Stop)

	m.handleCommand(context.Background(), Envelope{MessageId: "c1", Kind: KindCommand, Type: "unknown"})

	// Wait for the result envelope to be written.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		envs := conn.writtenEnvelopes()
		for _, env := range envs {
			if env.Kind == KindResult {
				var res CommandResult
				_ = json.Unmarshal(env.Payload, &res)
				assert.Equal(t, "c1", res.CommandId)
				assert.Equal(t, CmdStatusFailed, res.Status)
				assert.Contains(t, res.Error, "no handler")
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("never received KindResult envelope")
}
