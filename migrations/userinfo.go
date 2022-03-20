package migrations

const UserInfo = `
CREATE TABLE IF NOT EXISTS user_record (
	inner_id INTEGER PRIMARY KEY,
	id VARCHAR(255) NOT NULL,
	name VARCHAR(255) NOT NULL,
	session_token VARCHAR(255) NOT NULL,
	profile VARCHAR(255) NOT NULL,
	environment VARCHAR(10) NOT NULL,
	organization_id VARCHAR(255) NOT NULL,
	organization_name VARCHAR(255) NOT NULL,
	encryption_key VARCHAR(255) NOT NULL,
	updated_at TIMESTAMP NOT NULL
)
`

const UserSettings = `
CREATE TABLE IF NOT EXISTS user_settings (
	user_id VARCHAR(255) NOT NULL,
	profile VARCHAR(255) NOT NULL,
	use_dataset_id VARCHAR(255) NULL,
	PRIMARY KEY (user_id, profile)
)
`
