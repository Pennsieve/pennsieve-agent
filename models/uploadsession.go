package models

import (
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"log"
	"time"
)

type UploadSession struct {
	SessionId        string    `json:"session_id"`
	UserId           string    `json:"user_id"`
	UserName         string    `json:"user_name"`
	OrganizationId   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	DatasetId        string    `json:"dataset_id"`
	DatasetName      string    `json:"dataset_name"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UploadSessionParams struct {
	SessionId        string `json:"session_id"`
	UserId           string `json:"user_id"`
	UserName         string `json:"user_name"`
	OrganizationId   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	DatasetId        string `json:"dataset_id"`
	DatasetName      string `json:"dataset_name"`
}

// GetAll returns all rows in the Upload Record Table
func (*UploadSession) GetAll() ([]UploadSession, error) {
	rows, err := db.DB.Query("SELECT * FROM upload_sessions")
	var allSessions []UploadSession
	if err == nil {
		for rows.Next() {
			var currentRecord UploadSession
			err = rows.Scan(
				&currentRecord.SessionId,
				&currentRecord.UserId,
				&currentRecord.UserName,
				&currentRecord.OrganizationId,
				&currentRecord.OrganizationName,
				&currentRecord.DatasetId,
				&currentRecord.DatasetName,
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
	sqlInsert := "INSERT INTO upload_sessions(session_id, user_id, user_name, organization_id,  " +
		"organization_name, dataset_id, dataset_name, " +
		"status, created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)"
	stmt, err := db.DB.Prepare(sqlInsert)
	if err != nil {
		log.Fatalln("ERROR: ", err)
	}
	defer stmt.Close()

	indexStr := pb.ListManifestFilesResponse_INDEXED.String()

	// format all vals at once
	currentTime := time.Now()
	_, err = stmt.Exec(s.SessionId, s.UserId, s.UserName, s.OrganizationId, s.OrganizationName, s.DatasetId,
		s.DatasetName, indexStr, currentTime, currentTime)
	if err != nil {
		log.Println(err)
	}

	return err

}

func (*UploadSession) Remove(manifestId string) error {
	sqlDelete := "DELETE FROM upload_sessions WHERE session_id = ?"
	stmt, err := db.DB.Prepare(sqlDelete)
	if err != nil {
		log.Fatalln("ERROR: ", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(manifestId)

	if err != nil {
		log.Println(err)
	}

	return err
}
