package models

import (
	"github.com/pennsieve/pennsieve-agent/config"
	"log"
	"time"
)

type UploadSession struct {
	SessionId      string    `json:"session_id"`
	UserId         string    `json:"user_id"`
	OrganizationId string    `json:"organization_id"`
	DatasetId      string    `json:"dataset_id"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UploadSessionParams struct {
	SessionId      string `json:"session_id"`
	UserId         string `json:"user_id"`
	OrganizationId string `json:"organization_id"`
	DatasetId      string `json:"dataset_id"`
}

// GetAll returns all rows in the Upload Record Table
func (*UploadSession) GetAll() ([]UploadSession, error) {
	rows, err := config.DB.Query("SELECT * FROM upload_sessions")
	var allSessions []UploadSession
	if err == nil {
		for rows.Next() {
			var currentRecord UploadSession
			err = rows.Scan(
				&currentRecord.SessionId,
				&currentRecord.UserId,
				&currentRecord.OrganizationId,
				&currentRecord.DatasetId,
				&currentRecord.Status,
				&currentRecord.CreatedAt,
				&currentRecord.UpdatedAt)

			if err != nil {
				log.Println("ERROR: ", err)
			}

			allSessions = append(allSessions, currentRecord)
		}
		return allSessions, err
	}
	return allSessions, err
}

// Add adds multiple rows to the UploadRecords database.
func (*UploadSession) Add(s UploadSessionParams) error {
	sqlInsert := "INSERT INTO upload_sessions(session_id, user_id, organization_id, dataset_id, " +
		"status, created_at,updated_at) VALUES (?,?,?,?,?,?,?)"
	stmt, err := config.DB.Prepare(sqlInsert)
	if err != nil {
		log.Fatalln("ERROR: ", err)
	}
	defer stmt.Close()

	// format all vals at once
	currentTime := time.Now()
	_, err = stmt.Exec(s.SessionId, s.UserId, s.OrganizationId, s.DatasetId, "INITIALIZED", currentTime, currentTime)
	if err != nil {
		log.Println(err)
	}

	return nil

}
