package migrations

// UserInfo contains info about current user and session
// Multiple users can have concurrent sessions in the DB
const UserInfo = `
CREATE TABLE IF NOT EXISTS user_record (
	inner_id INTEGER PRIMARY KEY,
	id VARCHAR(255) NOT NULL,
	name VARCHAR(255) NOT NULL,
	session_token VARCHAR(255) NOT NULL,
	refresh_token VARCHAR(255) NOT NULL,
	token_expire TIMESTAMP NOT NULL,
	id_token VARCHAR(255) NOT NULL,
	profile VARCHAR(255) NOT NULL,
	environment VARCHAR(10) NOT NULL,
	organization_id VARCHAR(255) NOT NULL,
	organization_name VARCHAR(255) NOT NULL,
	updated_at TIMESTAMP NOT NULL
)
`

//UserSettings contains preferences for a particular user.
const UserSettings = `
CREATE TABLE IF NOT EXISTS user_settings (
	user_id VARCHAR(255) NOT NULL,
	profile VARCHAR(255) NOT NULL,
	use_dataset_id VARCHAR(255) NULL,
	PRIMARY KEY (user_id, profile)
)
`
