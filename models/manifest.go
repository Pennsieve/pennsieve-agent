package models

import (
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/manifest"
	"log"
	"time"
)

type Manifest struct {
	Id               int32           `json:"id"`
	NodeId           sql.NullString  `json:"node_id"`
	UserId           string          `json:"user_id"`
	UserName         string          `json:"user_name"`
	OrganizationId   string          `json:"organization_id"`
	OrganizationName string          `json:"organization_name"`
	DatasetId        string          `json:"dataset_id"`
	DatasetName      string          `json:"dataset_name"`
	Status           manifest.Status `json:"status"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type ManifestParams struct {
	UserId           string `json:"user_id"`
	UserName         string `json:"user_name"`
	OrganizationId   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	DatasetId        string `json:"dataset_id"`
	DatasetName      string `json:"dataset_name"`
}

// Get returns all rows in the Upload Record Table
func (*Manifest) Get(id int32) (*Manifest, error) {

	log.Println("Getting manifest with ID: ", id)

	var statusStr string
	res := &Manifest{}
	err := db.DB.QueryRow(fmt.Sprintf(
		"SELECT * FROM manifests WHERE id=%d", id)).Scan(
		&res.Id,
		&res.NodeId,
		&res.UserId,
		&res.UserName,
		&res.OrganizationId,
		&res.OrganizationName,
		&res.DatasetId,
		&res.DatasetName,
		&statusStr,
		&res.CreatedAt,
		&res.UpdatedAt)

	var m manifest.Status
	res.Status = m.ManifestStatusMap(statusStr)

	return res, err
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

			var m manifest.Status
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
		s.DatasetName, manifest.Initiated.String(), currentTime, currentTime).Scan(&id)
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
		Status:           manifest.Initiated,
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

// SetManifestNodeId updates the manifest Node ID in the Manifest object and Database
func (m *Manifest) SetManifestNodeId(nodeId string) error {

	m.NodeId = sql.NullString{
		String: nodeId,
		Valid:  true,
	}

	statement, err := db.DB.Prepare(
		"UPDATE manifests SET node_id = ? WHERE id = ?")
	if err != nil {
		log.Println(err)
		return err
	}

	_, err = statement.Exec(nodeId, m.Id)
	if err != nil {
		log.Println("Unable to update Manifest Node Id in database: ", err)
		return err
	}

	return nil
}

//func (*Manifest) SetStatus(manifestId int32, status)
