package models

import (
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	"log"
	"time"
)

type Manifest struct {
	Id               int32          `json:"id"`
	NodeId           sql.NullString `json:"node_id"`
	UserId           string         `json:"user_id"`
	UserName         string         `json:"user_name"`
	OrganizationId   string         `json:"organization_id"`
	OrganizationName string         `json:"organization_name"`
	DatasetId        string         `json:"dataset_id"`
	DatasetName      string         `json:"dataset_name"`
	Status           ManifestStatus `json:"status"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type ManifestParams struct {
	UserId           string `json:"user_id"`
	UserName         string `json:"user_name"`
	OrganizationId   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	DatasetId        string `json:"dataset_id"`
	DatasetName      string `json:"dataset_name"`
}

type ManifestStatus int64

const (
	ManifestInitiated ManifestStatus = iota
	ManifestUploading
	ManifestCompleted
	ManifestCancelled
)

func (s ManifestStatus) String() string {
	switch s {
	case ManifestInitiated:
		return "Initiated"
	case ManifestUploading:
		return "InProgress"
	case ManifestCompleted:
		return "Completed"
	case ManifestCancelled:
		return "Cancelled"
	default:
		return "Initiated"
	}
}

func (s ManifestStatus) ManifestStatusMap(value string) ManifestStatus {
	switch value {
	case "Initiated":
		return ManifestInitiated
	case "InProgress":
		return ManifestUploading
	case "Completed":
		return ManifestCompleted
	case "Cancelled":
		return ManifestCancelled
	}
	return ManifestInitiated
}

// Get returns all rows in the Upload Record Table
func (*Manifest) Get(id int32) (*Manifest, error) {

	var statusStr string
	manifest := &Manifest{}
	err := db.DB.QueryRow(fmt.Sprintf(
		"SELECT * FROM manifests WHERE id=%d", id)).Scan(
		&manifest.Id,
		&manifest.NodeId,
		&manifest.UserId,
		&manifest.UserName,
		&manifest.OrganizationId,
		&manifest.OrganizationName,
		&manifest.DatasetId,
		&manifest.DatasetName,
		&statusStr,
		&manifest.CreatedAt,
		&manifest.UpdatedAt)

	var m ManifestStatus
	manifest.Status = m.ManifestStatusMap(statusStr)

	return manifest, err
}

// GetAll returns all rows in the Upload Record Table
func (*Manifest) GetAll() ([]Manifest, error) {
	rows, err := db.DB.Query("SELECT * FROM manifests;")
	var allSessions []Manifest
	if err == nil {
		for rows.Next() {
			var statusStr string
			var currentRecord Manifest
			err = rows.Scan(
				&currentRecord.Id,
				&currentRecord.NodeId,
				&currentRecord.UserId,
				&currentRecord.UserName,
				&currentRecord.OrganizationId,
				&currentRecord.OrganizationName,
				&currentRecord.DatasetId,
				&currentRecord.DatasetName,
				&statusStr,
				&currentRecord.CreatedAt,
				&currentRecord.UpdatedAt)

			var m ManifestStatus
			currentRecord.Status = m.ManifestStatusMap(statusStr)

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
func (m *Manifest) Add(s ManifestParams) (*Manifest, error) {

	sqlStatement := "INSERT INTO manifests(user_id, user_name, organization_id,  " +
		"organization_name, dataset_id, dataset_name, " +
		"status, created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?) RETURNING id;"

	currentTime := time.Now()
	var id int32
	err := db.DB.QueryRow(sqlStatement, s.UserId, s.UserName, s.OrganizationId, s.OrganizationName, s.DatasetId,
		s.DatasetName, ManifestInitiated.String(), currentTime, currentTime).Scan(&id)
	if err != nil {
		panic(err)
	}

	createdManifest := Manifest{
		Id:               id,
		NodeId:           sql.NullString{},
		UserId:           s.UserId,
		UserName:         s.UserName,
		OrganizationId:   s.OrganizationId,
		OrganizationName: s.OrganizationName,
		DatasetId:        s.DatasetId,
		DatasetName:      s.DatasetName,
		Status:           ManifestInitiated,
		CreatedAt:        currentTime,
		UpdatedAt:        currentTime,
	}

	return &createdManifest, err
}

// Remove removes a manifest from the local DB.
func (*Manifest) Remove(manifestId int32) error {
	sqlDelete := "DELETE FROM manifests WHERE id = ?"
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

//func (*Manifest) SetStatus(manifestId int32, status)
