-- Local installation identity (one row per host).
CREATE TABLE IF NOT EXISTS cloud_installation (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    installation_id TEXT UNIQUE NOT NULL,
    created_at INTEGER NOT NULL
);

-- Per-profile registration with the Pennsieve Agent Service.
-- profile_name maps to the active Viper profile in config.ini.
CREATE TABLE IF NOT EXISTS cloud_registration (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_name TEXT UNIQUE NOT NULL,
    agent_id TEXT NOT NULL,
    agent_secret TEXT NOT NULL,
    ws_url TEXT NOT NULL,
    ws_token_url TEXT NOT NULL,
    heartbeat_sec INTEGER NOT NULL DEFAULT 270,
    registered_at INTEGER NOT NULL
);

-- Server-issued commands. The agent persists these locally so it can
-- reconcile state with the service after a crash or reconnect.
CREATE TABLE IF NOT EXISTS cloud_command (
    command_id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    payload TEXT NOT NULL,
    status TEXT NOT NULL,
    received_at INTEGER NOT NULL,
    started_at INTEGER,
    completed_at INTEGER,
    result TEXT,
    error TEXT
);

CREATE INDEX IF NOT EXISTS idx_cloud_command_status ON cloud_command(status);
