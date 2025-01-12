CREATE TABLE IF NOT EXISTS ts_channel (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id VARCHAR(255) UNIQUE NOT NULL,
    package_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    start_time INTEGER NOT NULL,
    end_time INTEGER NOT NULL,
    unit VARCHAR(255) NOT NULL,
    rate REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS ts_range (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id VARCHAR(255) UNIQUE NOT NULL,
    channel_node_id VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL,
    start_time INTEGER NOT NULL,
    end_time INTEGER NOT NULL,
    FOREIGN KEY(channel_node_id) REFERENCES ts_channel(node_id) ON DELETE CASCADE
)