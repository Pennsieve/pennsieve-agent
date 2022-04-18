package migrations

// UploadRecords Multiple users can have concurrent sessions in the DB
// Primary key: primary key in the DB
// Source_Path: location of file on local machine
// Target_Path: Optional path in Dataset on Pennsieve
// Dataset_ID: Dataset ID on Pennsieve
// Package_ID: Target Package ID on Pennsieve
// Import_ID: Import ID for file
// Import_Session_ID: ID for upload session
// Status: Upload status for file (LOCAL, PENDING, UPLOADING, COMPLETED)
const UploadRecords = `
CREATE TABLE IF NOT EXISTS upload_record (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	organization_id VARCHAR(255) NOT NULL,
	dataset_id VARCHAR(255) NOT NULL,
	package_id VARCHAR(255),
	source_path TEXT NOT NULL,
	target_path TEXT,
	import_id VARCHAR(255),
	import_session_id VARCHAR(255) NOT NULL,
	progress INTEGER,
	status VARCHAR(255) NOT NULL,
	created_at VARCHAR(255) NOT NULL,
	updated_at VARCHAR(255) NOT NULL
)
`
