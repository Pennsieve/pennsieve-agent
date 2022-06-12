package models

import (
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"log"
	"strings"
	"time"
)

type ManifestFile struct {
	Id         int32     `json:"id"`
	ManifestId int32     `json:"manifest_id"`
	SourcePath string    `json:"source_path"`
	TargetPath string    `json:"target_path"`
	S3Key      string    `json:"s3_key"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ManifestFileParams struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	S3Key      string `json:"s3_key"`
	ManifestId int32  `json:"manifest_id"`
}

func (*ManifestFile) Get(manifestId int32, limit int32, offset int32) ([]ManifestFile, error) {

	rows, err := db.DB.Query("SELECT * FROM manifest_files WHERE manifest_id = ? ORDER BY id LIMIT ? OFFSET ?",
		manifestId, limit, offset)
	if err != nil {
		return nil, err
	}

	var allRecords []ManifestFile
	for rows.Next() {
		var currentRecord ManifestFile
		err = rows.Scan(
			&currentRecord.Id,
			&currentRecord.ManifestId,
			&currentRecord.SourcePath,
			&currentRecord.TargetPath,
			&currentRecord.S3Key,
			&currentRecord.Status,
			&currentRecord.CreatedAt,
			&currentRecord.UpdatedAt)

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
			var currentRecord ManifestFile
			err = rows.Scan(
				&currentRecord.Id,
				&currentRecord.SourcePath,
				&currentRecord.TargetPath,
				&currentRecord.S3Key,
				&currentRecord.ManifestId,
				&currentRecord.Status,
				&currentRecord.CreatedAt,
				&currentRecord.UpdatedAt)

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

	currentTime := time.Now()
	const rowSQL = "(?, ?, ?, ?, ?, ?, ?)"
	var vals []interface{}
	var inserts []string
	indexStr := pb.ListManifestFilesResponse_INDEXED.String()

	sqlInsert := "INSERT INTO manifest_files(source_path, target_path, s3_key, " +
		"manifest_id, status, created_at, updated_at) VALUES "
	for _, row := range records {
		inserts = append(inserts, rowSQL)
		vals = append(vals, row.SourcePath, row.TargetPath, row.S3Key, row.ManifestId,
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
