package cloud

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestDB returns a fresh in-memory SQLite database with the cloud
// tables created. Each test gets its own DB so tests can run in
// parallel without colliding on schema state.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	schema := []string{
		`CREATE TABLE cloud_installation (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			installation_id TEXT UNIQUE NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE cloud_registration (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_name TEXT UNIQUE NOT NULL,
			agent_id TEXT NOT NULL,
			agent_secret TEXT NOT NULL,
			ws_url TEXT NOT NULL,
			ws_token_url TEXT NOT NULL,
			heartbeat_sec INTEGER NOT NULL DEFAULT 270,
			registered_at INTEGER NOT NULL
		)`,
		`CREATE TABLE cloud_command (
			command_id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			payload TEXT NOT NULL,
			status TEXT NOT NULL,
			received_at INTEGER NOT NULL,
			started_at INTEGER,
			completed_at INTEGER,
			result TEXT,
			error TEXT
		)`,
	}
	for _, s := range schema {
		_, err := db.Exec(s)
		require.NoError(t, err)
	}
	return db
}

func TestStore_Installation_GetOrCreateIsStable(t *testing.T) {
	s := NewStore(newTestDB(t))
	first, err := s.GetOrCreateInstallation(func() string { return "id-1" })
	require.NoError(t, err)
	assert.Equal(t, "id-1", first.InstallationId)

	// Second call returns the same row even if the factory would have
	// produced a different id.
	second, err := s.GetOrCreateInstallation(func() string { return "id-2" })
	require.NoError(t, err)
	assert.Equal(t, "id-1", second.InstallationId)
}

func TestStore_Registration_UpsertAndGet(t *testing.T) {
	s := NewStore(newTestDB(t))
	r := Registration{
		ProfileName:  "default",
		AgentId:      "a1",
		AgentSecret:  "s1",
		WSURL:        "wss://example",
		WSTokenURL:   "https://example/token",
		HeartbeatSec: 200,
		RegisteredAt: time.Unix(1700000000, 0),
	}
	require.NoError(t, s.UpsertRegistration(r))

	got, err := s.GetRegistration("default")
	require.NoError(t, err)
	assert.Equal(t, r.AgentId, got.AgentId)
	assert.Equal(t, r.AgentSecret, got.AgentSecret)

	// Upsert overwrites.
	r.AgentSecret = "s2"
	require.NoError(t, s.UpsertRegistration(r))
	got, _ = s.GetRegistration("default")
	assert.Equal(t, "s2", got.AgentSecret)
}

func TestStore_Registration_NotFound(t *testing.T) {
	s := NewStore(newTestDB(t))
	_, err := s.GetRegistration("missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestStore_Command_UpsertAndSeen(t *testing.T) {
	s := NewStore(newTestDB(t))
	now := time.Now()
	require.NoError(t, s.UpsertCommand(LocalCommand{
		CommandId:  "c1",
		Type:       "ping",
		Payload:    []byte(`{"x":1}`),
		Status:     CmdStatusReceived,
		ReceivedAt: now,
	}))
	seen, err := s.SeenCommand("c1")
	require.NoError(t, err)
	assert.True(t, seen)

	miss, err := s.SeenCommand("never")
	require.NoError(t, err)
	assert.False(t, miss)

	// Upserting with RUNNING preserves received_at and merges started_at.
	started := now.Add(time.Second)
	require.NoError(t, s.UpsertCommand(LocalCommand{
		CommandId: "c1",
		Type:      "ping",
		Payload:   []byte(`{"x":1}`),
		Status:    CmdStatusRunning,
		StartedAt: &started,
	}))
}

func TestStore_OutstandingCommands(t *testing.T) {
	s := NewStore(newTestDB(t))
	now := time.Now()

	cases := []struct {
		id, status string
	}{
		{"c-recv", CmdStatusReceived},
		{"c-run", CmdStatusRunning},
		{"c-done", CmdStatusCompleted},
		{"c-fail", CmdStatusFailed},
	}
	for _, c := range cases {
		require.NoError(t, s.UpsertCommand(LocalCommand{
			CommandId: c.id, Type: "x", Payload: []byte("{}"),
			Status: c.status, ReceivedAt: now,
		}))
	}

	out, err := s.OutstandingCommands()
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, c := range out {
		ids[c.CommandId] = true
	}
	assert.True(t, ids["c-recv"])
	assert.True(t, ids["c-run"])
	assert.False(t, ids["c-done"])
	assert.False(t, ids["c-fail"])
}
