-- UserInfo contains info about current user and session
-- Multiple users can have concurrent sessions in the DB
CREATE TABLE IF NOT EXISTS user_record (
    inner_id          INTEGER PRIMARY KEY,
    id                VARCHAR(255) NOT NULL,
    name              VARCHAR(255) NOT NULL,
    session_token     VARCHAR(255) NOT NULL,
    refresh_token     VARCHAR(255) NOT NULL,
    token_expire      TIMESTAMP    NOT NULL,
    id_token          VARCHAR(255) NOT NULL,
    profile           VARCHAR(255) NOT NULL,
    environment       VARCHAR(10)  NOT NULL,
    organization_id   VARCHAR(255) NOT NULL,
    organization_name VARCHAR(255) NOT NULL,
    updated_at        TIMESTAMP    NOT NULL
);

-- UserSettings contains preferences for a particular user.
CREATE TABLE IF NOT EXISTS user_settings (
     user_id VARCHAR(255) NOT NULL,
     profile VARCHAR(255) NOT NULL,
     use_dataset_id VARCHAR(255) NULL,
     PRIMARY KEY (user_id, profile)
);

-- ManifestFiles Multiple users can have concurrent sessions in the DB
-- Primary key: primary key in the DB
-- Source_Path: location of file on local machine
-- Target_Path: Optional path in Dataset on Pennsieve
-- Session_ID: ID for upload session
-- Status: Upload status for file (LOCAL, PENDING, UPLOADING, COMPLETED, CANCELED)
CREATE TABLE IF NOT EXISTS manifest_files (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      manifest_id INTEGER NOT NULL,
      upload_id VARCHAR(255) NOT NULL,
      source_path TEXT NOT NULL,
      target_path TEXT,
      target_name VARCHAR(255) NOT NULL,
      status VARCHAR(255) NOT NULL,
      created_at TIMESTAMP NOT NULL,
      updated_at TIMESTAMP NOT NULL,
      CONSTRAINT fk_manifest_id
          FOREIGN KEY (manifest_id)
              REFERENCES manifests(id)
              ON DELETE CASCADE
);

-- Manifests is a table that tracks locally initiated upload sessions
-- User_id: Pennsieve user id for user initiating the session
-- Organization_id: Organization id for upload session
-- Dataset_id: Dataset id for upload session
-- Status: Upload status for file (INITIATED, IN_PROGRESS, COMPLETED, CANCELED)
--
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
);

