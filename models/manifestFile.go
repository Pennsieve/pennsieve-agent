package models

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/pennsieve/pennsieve-go-api/models/manifest/manifestFile"
	"log"
	"strings"
	"time"
)

type ManifestFile struct {
	Id         int32               `json:"id"`
	ManifestId int32               `json:"manifest_id"`
	UploadId   uuid.UUID           `json:"upload_id"`
	SourcePath string              `json:"source_path"`
	TargetPath string              `json:"target_path"`
	TargetName string              `json:"target_name"`
	Status     manifestFile.Status `json:"status"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
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

		var s manifestFile.Status
		currentRecord.Status = s.ManifestFileStatusMap(status)

		if err != nil {
			log.Println("ERROR: ", err)
		}

		allRecords = append(allRecords, currentRecord)
	}
	return allRecords, err

}

// GetByStatus returns files in a manifest filtered by status.
func (*ManifestFile) GetByStatus(manifestId int32, statusArray []manifestFile.Status, limit int, offset int) ([]ManifestFile, error) {

	var statusList []string
	for _, reqStatus := range statusArray {
		statusList = append(statusList, fmt.Sprintf("'%s'", reqStatus.String()))
	}
	statusQueryString := fmt.Sprintf("(%s)", strings.Join(statusList, ","))
	queryStr := fmt.Sprintf("SELECT * FROM manifest_files WHERE manifest_id = ? "+
		"AND status IN %s ORDER BY id LIMIT ? OFFSET ?", statusQueryString)

	log.Println(queryStr)
	rows, err := db.DB.Query(queryStr, manifestId, limit, offset)
	if err != nil {
		log.Println("Error getting rows from Manifest Files:", err)
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

		var s manifestFile.Status
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

			var s manifestFile.Status
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
	indexStr := manifestFile.Initiated.String()

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

func (*ManifestFile) SyncResponseStatusUpdate(manifestId int32, statusList []manifestFile.FileStatusDTO) error {

	allStatus := []manifestFile.Status{
		manifestFile.Initiated,
		manifestFile.Synced,
		manifestFile.Imported,
		manifestFile.Finalized,
		manifestFile.Verified,
		manifestFile.Unknown,
		manifestFile.Failed,
		manifestFile.Removed,
	}

	idByStatus := map[string][]string{}
	// create map from array
	for _, s := range allStatus {
		statusString := s.String()
		for _, f := range statusList {
			if f.Status == s {
				idByStatus[statusString] = append(idByStatus[statusString], fmt.Sprintf("'%s'", f.UploadId))
			}
		}
	}

	// Iterate over map and update SQL
	for key, s := range idByStatus {
		if len(s) > 0 {
			allUploadIds := strings.Join(s, ",")

			sqlStatement := fmt.Sprintf("UPDATE manifest_files SET status = '%s' "+
				"WHERE manifest_id = %d AND upload_id IN (%s);", key, manifestId, allUploadIds)

			log.Println(sqlStatement)

			_, err := db.DB.Exec(sqlStatement)
			if err != nil {
				log.Println("Unable to update status in manifest files for manifest:", manifestId, "--", err)
				return err
			}

		}

	}

	return nil
	//

}

// SyncResponseStatusUpdate2 updates local DB based on successful/unsuccessful updates remotely.
// 1. Set to SYNCED for all files that were successfully synchronized (Initiated, Failed)
// 2. Remove files with REMOVED that were successfully removed remotely.
func (*ManifestFile) SyncResponseStatusUpdate2(manifestId int32, failedFiles []string) {

	// Set INITIATED and FAILED to SYNCED
	requestStatus := []manifestFile.Status{
		manifestFile.Initiated,
		manifestFile.Failed,
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
		manifestFile.Synced.String(), manifestId, statusQueryString)

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
		manifestId, manifestFile.Removed.String())

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
	initatedStatus := manifestFile.Initiated.String()
	queryStr := fmt.Sprintf("DELETE FROM manifest_files WHERE manifest_id = %d "+
		"AND source_path LIKE %s and status = '%s';", manifestId, pathLikeExpr, initatedStatus)

	log.Println(queryStr)

	_, err := db.DB.Exec(queryStr)
	if err != nil {
		return err
	}

	syncStatus := manifestFile.Synced.String()
	removeStatus := manifestFile.Removed.String()
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

func (*ManifestFile) ResetStatusForManifest(manifestId int32) error {

	log.Println("IN RESET MANIFEST")
	currentTime := time.Now()

	initiatedStatusStr := manifestFile.Initiated.String()
	sqlStatement := fmt.Sprintf("UPDATE manifest_files SET status = '%s', updated_at = %d WHERE manifest_id = %d",
		initiatedStatusStr, currentTime.Unix(), manifestId)

	log.Println(sqlStatement)
	// format all vals at once
	_, err := db.DB.Exec(sqlStatement)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
