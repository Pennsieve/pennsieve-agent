package cloud

import (
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

// Installation is the host's local identity. Stable across restarts;
// regenerated only if the SQLite database is wiped.
type Installation struct {
	InstallationId string
	CreatedAt      time.Time
}

// Registration is per-profile. profileName matches the Viper profile
// section in ~/.pennsieve/config.ini.
type Registration struct {
	ProfileName  string
	AgentId      string
	AgentSecret  string
	WSURL        string
	WSTokenURL   string
	HeartbeatSec int
	RegisteredAt time.Time
}

// LocalCommand mirrors the server-side command record so the agent can
// recover state on restart and reconcile with the service.
type LocalCommand struct {
	CommandId   string
	Type        string
	Payload     []byte
	Status      string
	ReceivedAt  time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Result      []byte
	Error       string
}

type Store struct {
	DB *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{DB: db} }

// GetOrCreateInstallation returns the local installation, creating it
// (with a fresh UUID) on first call. Caller supplies the UUID factory
// so tests can be deterministic.
func (s *Store) GetOrCreateInstallation(newID func() string) (Installation, error) {
	row := s.DB.QueryRow(`SELECT installation_id, created_at FROM cloud_installation LIMIT 1`)
	var inst Installation
	var ts int64
	if err := row.Scan(&inst.InstallationId, &ts); err == nil {
		inst.CreatedAt = time.Unix(ts, 0)
		return inst, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Installation{}, err
	}
	id := newID()
	now := time.Now().Unix()
	if _, err := s.DB.Exec(`INSERT INTO cloud_installation (installation_id, created_at) VALUES (?, ?)`, id, now); err != nil {
		return Installation{}, err
	}
	return Installation{InstallationId: id, CreatedAt: time.Unix(now, 0)}, nil
}

func (s *Store) GetRegistration(profile string) (Registration, error) {
	row := s.DB.QueryRow(`
		SELECT profile_name, agent_id, agent_secret, ws_url, ws_token_url, heartbeat_sec, registered_at
		FROM cloud_registration WHERE profile_name = ?`, profile)
	var r Registration
	var ts int64
	if err := row.Scan(&r.ProfileName, &r.AgentId, &r.AgentSecret, &r.WSURL, &r.WSTokenURL, &r.HeartbeatSec, &ts); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Registration{}, ErrNotFound
		}
		return Registration{}, err
	}
	r.RegisteredAt = time.Unix(ts, 0)
	return r, nil
}

func (s *Store) UpsertRegistration(r Registration) error {
	_, err := s.DB.Exec(`
		INSERT INTO cloud_registration
			(profile_name, agent_id, agent_secret, ws_url, ws_token_url, heartbeat_sec, registered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(profile_name) DO UPDATE SET
			agent_id      = excluded.agent_id,
			agent_secret  = excluded.agent_secret,
			ws_url        = excluded.ws_url,
			ws_token_url  = excluded.ws_token_url,
			heartbeat_sec = excluded.heartbeat_sec,
			registered_at = excluded.registered_at`,
		r.ProfileName, r.AgentId, r.AgentSecret, r.WSURL, r.WSTokenURL, r.HeartbeatSec, r.RegisteredAt.Unix())
	return err
}

func (s *Store) UpsertCommand(c LocalCommand) error {
	var startedAt, completedAt sql.NullInt64
	if c.StartedAt != nil {
		startedAt = sql.NullInt64{Int64: c.StartedAt.Unix(), Valid: true}
	}
	if c.CompletedAt != nil {
		completedAt = sql.NullInt64{Int64: c.CompletedAt.Unix(), Valid: true}
	}
	// Payload is NOT NULL in the schema; coalesce nil to empty bytes
	// so callers updating an existing command (without re-supplying
	// the payload) don't trip the constraint on the INSERT side of
	// the upsert.
	payload := c.Payload
	if payload == nil {
		payload = []byte{}
	}
	_, err := s.DB.Exec(`
		INSERT INTO cloud_command
			(command_id, type, payload, status, received_at, started_at, completed_at, result, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(command_id) DO UPDATE SET
			status       = excluded.status,
			started_at   = COALESCE(excluded.started_at, cloud_command.started_at),
			completed_at = COALESCE(excluded.completed_at, cloud_command.completed_at),
			result       = COALESCE(excluded.result, cloud_command.result),
			error        = excluded.error`,
		c.CommandId, c.Type, payload, c.Status, c.ReceivedAt.Unix(),
		startedAt, completedAt, c.Result, c.Error)
	return err
}

// SeenCommand returns true if the agent has previously recorded this
// commandId. Used for at-least-once idempotency: a second delivery of
// the same command is a no-op.
func (s *Store) SeenCommand(commandId string) (bool, error) {
	row := s.DB.QueryRow(`SELECT 1 FROM cloud_command WHERE command_id = ?`, commandId)
	var x int
	err := row.Scan(&x)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// OutstandingCommands returns commands that have not reached a terminal
// state. Used for reconciliation on reconnect.
func (s *Store) OutstandingCommands() ([]LocalCommand, error) {
	rows, err := s.DB.Query(`
		SELECT command_id, type, payload, status, received_at,
		       started_at, completed_at, result, error
		FROM cloud_command
		WHERE status IN (?, ?)`,
		CmdStatusReceived, CmdStatusRunning)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LocalCommand
	for rows.Next() {
		var c LocalCommand
		var receivedAt int64
		var startedAt, completedAt sql.NullInt64
		var errStr sql.NullString
		if err := rows.Scan(&c.CommandId, &c.Type, &c.Payload, &c.Status, &receivedAt,
			&startedAt, &completedAt, &c.Result, &errStr); err != nil {
			return nil, err
		}
		c.ReceivedAt = time.Unix(receivedAt, 0)
		if startedAt.Valid {
			t := time.Unix(startedAt.Int64, 0)
			c.StartedAt = &t
		}
		if completedAt.Valid {
			t := time.Unix(completedAt.Int64, 0)
			c.CompletedAt = &t
		}
		if errStr.Valid {
			c.Error = errStr.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
