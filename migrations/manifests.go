package migrations

// ManifestFiles Multiple users can have concurrent sessions in the DB
// Primary key: primary key in the DB
// Source_Path: location of file on local machine
// Target_Path: Optional path in Dataset on Pennsieve
// Session_ID: ID for upload session
// Status: Upload status for file (LOCAL, PENDING, UPLOADING, COMPLETED, CANCELED)
const ManifestFiles = `
CREATE TABLE IF NOT EXISTS manifest_files (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	manifest_id INTEGER NOT NULL,
	source_path TEXT NOT NULL,
	target_path TEXT,
	s3_key TEXT NOT NULL,
	status VARCHAR(255) NOT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	CONSTRAINT fk_manifest_id
		FOREIGN KEY (manifest_id)
		REFERENCES manifests(id)
		ON DELETE CASCADE
)
`

// Manifests is a table that tracks locally initiated upload sessions
// User_id: Pennsieve user id for user initiating the session
// Organization_id: Organization id for upload session
// Dataset_id: Dataset id for upload session
// Status: Upload status for file (INITIATED, IN_PROGRESS, COMPLETED, CANCELED)
const Manifests = `
CREATE TABLE IF NOT EXISTS manifests (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	node_id VARCHAR(255),
	user_id VARCHAR(255) NOT NULL,
	user_name VARCHAR(255) NOT NULL,
	organization_id VARCHAR(255) NOT NULL,
	organization_name VARCHAR(255) NOT NULL,
	dataset_id VARCHAR(255) NOT NULL,
	dataset_name VARCHAR(255) NOT NULL,
	status VARCHAR(255) NOT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL
)
`
