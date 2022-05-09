package models

import (
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"log"
	"strings"
	"time"
)

type UploadRecord struct {
	Id         int       `json:"id"`
	SourcePath string    `json:"source_path"`
	TargetPath string    `json:"target_path"`
	S3Key      string    `json:"s3_key"`
	SessionID  string    `json:"session_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UploadRecordParams struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	S3Key      string `json:"s3_key"`
	SessionID  string `json:"session_id"`
}

func (*UploadRecord) Get(manifestId string, limit int32, offset int32) ([]UploadRecord, error) {

	rows, err := db.DB.Query("SELECT * FROM upload_record WHERE session_id = ? ORDER BY id LIMIT ? OFFSET ?",
		manifestId, limit, offset)
	if err != nil {
		return nil, err
	}

	var allRecords []UploadRecord
	for rows.Next() {
		var currentRecord UploadRecord
		err = rows.Scan(
			&currentRecord.Id,
			&currentRecord.SourcePath,
			&currentRecord.TargetPath,
			&currentRecord.S3Key,
			&currentRecord.SessionID,
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
func (*UploadRecord) GetAll() ([]UploadRecord, error) {
	rows, err := db.DB.Query("SELECT * FROM upload_record")
	var allRecords []UploadRecord
	if err == nil {
		for rows.Next() {
			var currentRecord UploadRecord
			err = rows.Scan(
				&currentRecord.Id,
				&currentRecord.SourcePath,
				&currentRecord.TargetPath,
				&currentRecord.S3Key,
				&currentRecord.SessionID,
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
func (*UploadRecord) Add(records []UploadRecordParams) error {

	currentTime := time.Now()
	const rowSQL = "(?, ?, ?, ?, ?, ?, ?)"
	var vals []interface{}
	var inserts []string
	indexStr := pb.ListFilesResponse_INDEXED.String()

	sqlInsert := "INSERT INTO upload_record(source_path, target_path, s3_key, " +
		"session_id, status, created_at, updated_at) VALUES "
	for _, row := range records {
		inserts = append(inserts, rowSQL)
		vals = append(vals, row.SourcePath, row.TargetPath, row.S3Key, row.SessionID,
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

// TODO: Remove uploadsession

// TODO:
