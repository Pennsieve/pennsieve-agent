CREATE TABLE IF NOT EXISTS ts_channel (
    inner_id INTEGER PRIMARY KEY,
    node_id VARCHAR(255) NOT NULL,
    package_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    start_time INTEGER NOT NULL,
    end_time INTEGER NOT NULL,
    unit VARCHAR(255) NOT NULL,
    rate REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS ts_range (
    inner_id INTEGER PRIMARY KEY,
    id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    package_id VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL,
    start_time INTEGER NOT NULL,
    end_time INTEGER NOT NULL,
    FOREIGN KEY(channel_id) REFERENCES ts_channel(id) ON DELETE CASCADE
)