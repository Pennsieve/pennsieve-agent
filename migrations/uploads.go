package migrations

// UploadRecords Multiple users can have concurrent sessions in the DB
// Primary key: primary key in the DB
// Source_Path: location of file on local machine
// Target_Path: Optional path in Dataset on Pennsieve
// Import_Session_ID: ID for upload session
// Status: Upload status for file (LOCAL, PENDING, UPLOADING, COMPLETED, CANCELED)
const UploadRecords = `
CREATE TABLE IF NOT EXISTS upload_record (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	source_path TEXT NOT NULL,
	target_path TEXT,
	import_session_id VARCHAR(255) NOT NULL,
	status VARCHAR(255) NOT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL
)
`

// UploadSessions is a table that tracks locally initiated upload sessions
// User_id: Pennsieve user id for user initiating the session
// Organization_id: Organization id for upload session
// Dataset_id: Dataset id for upload session
// Status: Upload status for file (INITIATED, IN_PROGRESS, COMPLETED, CANCELED)
const UploadSessions = `
CREATE TABLE IF NOT EXISTS upload_sessions (
	session_id VARCHAR(255) PRIMARY KEY,
	user_id VARCHAR(255) NOT NULL,
	organization_id VARCHAR(255) NOT NULL,
	dataset_id VARCHAR(255) NOT NULL,
	status VARCHAR(255) NOT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL
)
`
