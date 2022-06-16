package models

import (
	"github.com/google/uuid"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/pennsieve/pennsieve-go-api/models/manifest"
	"log"
	"strings"
	"time"
)

type ManifestFile struct {
	Id         int32                       `json:"id"`
	ManifestId int32                       `json:"manifest_id"`
	UploadId   uuid.UUID                   `json:"upload_id""`
	SourcePath string                      `json:"source_path"`
	TargetPath string                      `json:"target_path"`
	Status     manifest.ManifestFileStatus `json:"status"`
	CreatedAt  time.Time                   `json:"created_at"`
	UpdatedAt  time.Time                   `json:"updated_at"`
}

type ManifestFileParams struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	ManifestId int32  `json:"manifest_id"`
}

func (*ManifestFile) Get(manifestId int32, limit int32, offset int32) ([]ManifestFile, error) {

	rows, err := db.DB.Query("SELECT * FROM manifest_files WHERE manifest_id = ? ORDER BY id LIMIT ? OFFSET ?",
		manifestId, limit, offset)
	if err != nil {
		return nil, err
	}

	var status string
	var allRecords []ManifestFile
	for rows.Next() {
		var currentRecord ManifestFile
		err = rows.Scan(
			&currentRecord.Id,
			&currentRecord.ManifestId,
			&currentRecord.UploadId,
			&currentRecord.SourcePath,
			&currentRecord.TargetPath,
			&status,
			&currentRecord.CreatedAt,
			&currentRecord.UpdatedAt)

		var s manifest.ManifestFileStatus
		currentRecord.Status = s.ManifestFileStatusMap(status)

		if err != nil {
			log.Println("ERROR: ", err)
		}

		allRecords = append(allRecords, currentRecord)
	}
	return allRecords, err

}

// GetAll returns all rows in the Upload Record Table
func (*ManifestFile) GetAll() ([]ManifestFile, error) {
	rows, err := db.DB.Query("SELECT * FROM manifest_files")
	var allRecords []ManifestFile
	if err == nil {
		for rows.Next() {
			var status string
			var currentRecord ManifestFile
			err = rows.Scan(
				&currentRecord.Id,
				&currentRecord.ManifestId,
				&currentRecord.UploadId,
				&currentRecord.SourcePath,
				&currentRecord.TargetPath,
				&status,
				&currentRecord.CreatedAt,
				&currentRecord.UpdatedAt)

			var s manifest.ManifestFileStatus
			currentRecord.Status = s.ManifestFileStatusMap(status)

			if err != nil {
				log.Println("ERROR: ", err)
			}

			allRecords = append(allRecords, currentRecord)
		}
		return allRecords, err
	}
	return allRecords, err
}

// Add adds multiple rows to the UploadRecords database.
func (*ManifestFile) Add(records []ManifestFileParams) error {

	uploadId := uuid.New()
	currentTime := time.Now()
	const rowSQL = "(?, ?, ?, ?, ?, ?, ?)"
	var vals []interface{}
	var inserts []string
	indexStr := pb.ListManifestFilesResponse_INDEXED.String()

	sqlInsert := "INSERT INTO manifest_files(source_path, target_path, upload_id, " +
		"manifest_id, status, created_at, updated_at) VALUES "
	for _, row := range records {
		inserts = append(inserts, rowSQL)
		vals = append(vals, row.SourcePath, row.TargetPath, uploadId.String(), row.ManifestId,
			indexStr, currentTime, currentTime)
	}
	sqlInsert = sqlInsert + strings.Join(inserts, ",")

	//prepare the statement
	stmt, err := db.DB.Prepare(sqlInsert)
	if err != nil {
		log.Fatalln("ERROR: ", err)
	}
	defer stmt.Close()

	// format all vals at once
	_, err = stmt.Exec(vals...)
	if err != nil {
		log.Println(err)
	}

	return nil

}
