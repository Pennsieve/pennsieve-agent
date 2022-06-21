package models

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/pennsieve/pennsieve-go-api/models/manifest"
	"log"
	"strings"
	"time"
)

type ManifestFile struct {
	Id         int32                       `json:"id"`
	ManifestId int32                       `json:"manifest_id"`
	UploadId   uuid.UUID                   `json:"upload_id"`
	SourcePath string                      `json:"source_path"`
	TargetPath string                      `json:"target_path"`
	TargetName string                      `json:"target_name"`
	Status     manifest.ManifestFileStatus `json:"status"`
	CreatedAt  time.Time                   `json:"created_at"`
	UpdatedAt  time.Time                   `json:"updated_at"`
}

type ManifestFileParams struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	TargetName string `json:"target_name"`
	ManifestId int32  `json:"manifest_id"`
}

// Get returns manifest paginated manifest files.
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
			&currentRecord.TargetName,
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

// GetByStatus returns files in a manifest filtered by status.
func (*ManifestFile) GetByStatus(manifestId int32, statusArray []manifest.ManifestFileStatus, limit int32, offset int32) ([]ManifestFile, error) {

	var statusList []string
	for _, reqStatus := range statusArray {
		statusList = append(statusList, fmt.Sprintf("'%s'", reqStatus.String()))
	}
	statusQueryString := fmt.Sprintf("(%s)", strings.Join(statusList, ","))
	queryStr := fmt.Sprintf("SELECT * FROM manifest_files WHERE manifest_id = ? "+
		"AND status IN %s ORDER BY id LIMIT ? OFFSET ?", statusQueryString)

	rows, err := db.DB.Query(queryStr, manifestId, limit, offset)
	if err != nil {
		log.Println("adsdsadsadasdsa", err)
		return nil, err
	}

	var st string
	var allRecords []ManifestFile
	for rows.Next() {
		var currentRecord ManifestFile
		err = rows.Scan(
			&currentRecord.Id,
			&currentRecord.ManifestId,
			&currentRecord.UploadId,
			&currentRecord.SourcePath,
			&currentRecord.TargetPath,
			&currentRecord.TargetName,
			&st,
			&currentRecord.CreatedAt,
			&currentRecord.UpdatedAt)

		var s manifest.ManifestFileStatus
		currentRecord.Status = s.ManifestFileStatusMap(st)

		if err != nil {
			log.Println("ERROR: ", err)
		}

		allRecords = append(allRecords, currentRecord)
	}

	log.Println("NUMBER OF RECORDS: ", len(allRecords))
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
				&currentRecord.TargetName,
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

	currentTime := time.Now()
	const rowSQL = "(?, ?, ?, ?, ?, ?, ?, ?)"
	var vals []interface{}
	var inserts []string
	indexStr := manifest.FileInitiated.String()

	sqlInsert := "INSERT INTO manifest_files(source_path, target_path, target_name, upload_id, " +
		"manifest_id, status, created_at, updated_at) VALUES "
	for _, row := range records {
		uploadId := uuid.New()
		inserts = append(inserts, rowSQL)
		vals = append(vals, row.SourcePath, row.TargetPath, row.TargetName, uploadId.String(), row.ManifestId,
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

// SyncResponseStatusUpdate updates local DB based on successful/unsuccessful updates remotely.
// 1. Set to SYNCED for all files that were successfully synchronized (Initiated, Failed)
// 2. Remove files with REMOVED that were successfully removed remotely.
func (*ManifestFile) SyncResponseStatusUpdate(manifestId int32, failedFiles []string) {

	// Set INITIATED and FAILED to SYNCED
	requestStatus := []manifest.ManifestFileStatus{
		manifest.FileInitiated,
		manifest.FileFailed,
	}

	var statusList []string
	for _, reqStatus := range requestStatus {
		statusList = append(statusList, fmt.Sprintf("'%s'", reqStatus.String()))
	}
	statusQueryString := fmt.Sprintf("(%s)", strings.Join(statusList, ","))

	var failedList []string
	for _, fileUploadId := range failedFiles {
		failedList = append(failedList, fmt.Sprintf("'%s'", fileUploadId))
	}

	queryString := fmt.Sprintf("UPDATE manifest_files SET status = '%s' WHERE manifest_id = %d AND status in %s",
		manifest.FileSynced.String(), manifestId, statusQueryString)

	if len(failedList) > 0 {
		failedFilesString := fmt.Sprintf("(%s)", strings.Join(failedList, ","))
		queryString = queryString + fmt.Sprintf(" AND NOT IN %s", failedFilesString)

	}
	queryString = queryString + ";"

	stmt, err := db.DB.Prepare(queryString)
	if err != nil {
		log.Fatalln("ERROR: ", err)
	}
	//defer stmt.Close()

	log.Println(stmt)

	// format all vals at once
	_, err = stmt.Exec()
	if err != nil {
		log.Println(err)
	}

	stmt.Close()

	// REMOVE FILES THAT WERE SUCCESSFULLY REMOVED
	queryString = fmt.Sprintf("DELETE FROM manifest_files WHERE manifest_id = %d AND status = '%s'",
		manifestId, manifest.FileRemoved.String())

	if len(failedList) > 0 {
		failedFilesString := fmt.Sprintf("(%s)", strings.Join(failedList, ","))
		queryString = queryString + fmt.Sprintf(" AND NOT IN %s", failedFilesString)

	}
	queryString = queryString + ";"

	stmt, err = db.DB.Prepare(queryString)
	if err != nil {
		log.Fatalln("ERROR: ", err)
	}
	//defer stmt.Close()

	log.Println(stmt)

	// format all vals at once
	_, err = stmt.Exec()
	if err != nil {
		log.Println(err)
	}
	stmt.Close()

	/*
			FileInitiated
			FileSynced
			FileUploaded
			FileVerified
			FileFailed
			FileRemoved

			New file --> Initiated

			update file (Initiated) --> Initiated
		 	update file (Synced) --> Updated
			update file (Removed) --> Initiated
			update file (FAILED) --> Updated

			remove file (Initiated) --> remove file
			remove file (Synced) --> REMOVED
			remove file (Failed) --> remove file

			sync file (Initiated) --> Synced
		    sync file (Removed) --> remove file
			sync file (Failed) --> Synced
			sync file but failed (Initialized, Failed) --> Failed
			sync file but failed (Removed) --> Removed

			status file (Synced) --> Completed
			status file (*) --> Some sort of error
	*/

}

// RemoveFromManifest removes files from local manifest by path.
func (*ManifestFile) RemoveFromManifest(manifestId int32, removePath string) error {

	pathLikeExpr := fmt.Sprintf("'%s%%'", removePath)
	initatedStatus := manifest.FileInitiated.String()
	queryStr := fmt.Sprintf("DELETE FROM manifest_files WHERE manifest_id = %d "+
		"AND source_path LIKE %s and status = '%s';", manifestId, pathLikeExpr, initatedStatus)

	log.Println(queryStr)

	_, err := db.DB.Exec(queryStr)
	if err != nil {
		return err
	}

	syncStatus := manifest.FileSynced.String()
	removeStatus := manifest.FileRemoved.String()
	queryStr2 := fmt.Sprintf("UPDATE manifest_files SET status = '%s' WHERE manifest_id = %d "+
		"AND source_path LIKE %s and status = '%s';", removeStatus, manifestId, pathLikeExpr, syncStatus)

	_, err = db.DB.Exec(queryStr2)
	if err != nil {
		return err
	}

	return nil
}

// ReplacePath allows users to replace the target path with another path.
func (*ManifestFile) ReplacePath(manifestId int32, path string, replacePath string, fileIds []int32) {

	//if replacePath[len(replacePath)-1:] != "\\" {
	//	replacePath = replacePath + "\\"
	//}
	//
	//pathLikeExpr := fmt.Sprintf("'%s%%'", removePath)
	//initatedStatus := manifest.FileInitiated.String()
	//
	//
	//queryStr := fmt.Sprintf("UPDATE manifest_files SET target_path = REPLACE(target_path, %s, %s) AND status ")
	//
	//queryStr := fmt.Sprintf("DELETE FROM manifest_files WHERE manifest_id = %d "+
	//	"AND source_path LIKE %s and status = '%s';", manifestId, pathLikeExpr, initatedStatus)
	//
	//log.Println(queryStr)

}
