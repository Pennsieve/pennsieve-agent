package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/manifest/manifestFile"
	log "github.com/sirupsen/logrus"
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

type ManifestFileStore interface {
	Get(manifestId int32, limit int32, offset int32) ([]ManifestFile, error)
	GetByStatus(manifestId int32, statusArray []manifestFile.Status, limit int, offset int) ([]ManifestFile, error)
	Add(records []ManifestFileParams) error
	BatchSetStatus(status manifestFile.Status, uploadIds []string) error
	SetStatus(status manifestFile.Status, uploadId string) error
	SyncResponseStatusUpdate(manifestId int32, statusList []manifestFile.FileStatusDTO) error
	SyncResponseStatusUpdate2(manifestId int32, failedFiles []string)
	RemoveFromManifest(manifestId int32, removePath string) error
	ResetStatusForManifest(manifestId int32) error
	GetNumberOfRowsForStatus(manifestId int32, statusArr []manifestFile.Status, invert bool) (int64, error)
	ManifestFilesToChannel(ctx context.Context, manifestId int32, statusArr []manifestFile.Status, walker chan<- ManifestFile)
}

func NewManifestFileStore(db *sql.DB) *manifestFileStore {
	return &manifestFileStore{
		db: db,
	}
}

type manifestFileStore struct {
	db *sql.DB
}

// Get returns manifest paginated manifest files.
func (s *manifestFileStore) Get(manifestId int32, limit int32, offset int32) ([]ManifestFile, error) {

	rows, err := s.db.Query("SELECT * FROM manifest_files WHERE manifest_id = ? ORDER BY id LIMIT ? OFFSET ?",
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
			log.Error("ERROR: ", err)
		}

		allRecords = append(allRecords, currentRecord)
	}
	return allRecords, err

}

// GetByStatus returns files in a manifest filtered by status.
func (s *manifestFileStore) GetByStatus(manifestId int32, statusArray []manifestFile.Status, limit int, offset int) ([]ManifestFile, error) {

	var statusList []string
	for _, reqStatus := range statusArray {
		statusList = append(statusList, fmt.Sprintf("'%s'", reqStatus.String()))
	}
	statusQueryString := fmt.Sprintf("(%s)", strings.Join(statusList, ","))
	queryStr := fmt.Sprintf("SELECT * FROM manifest_files WHERE manifest_id = ? "+
		"AND status IN %s ORDER BY id LIMIT ? OFFSET ?", statusQueryString)

	log.Debug(queryStr)
	rows, err := s.db.Query(queryStr, manifestId, limit, offset)
	if err != nil {
		log.Error("Error getting rows from Manifest Files:", err)
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
			log.Error("ERROR: ", err)
		}

		allRecords = append(allRecords, currentRecord)
	}

	return allRecords, err

}

// Add adds multiple rows to the UploadRecords database.
func (s *manifestFileStore) Add(records []ManifestFileParams) error {

	currentTime := time.Now()
	const rowSQL = "(?, ?, ?, ?, ?, ?, ?, ?)"
	var vals []interface{}
	var inserts []string
	indexStr := manifestFile.Local.String()

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
	stmt, err := s.db.Prepare(sqlInsert)
	if err != nil {
		log.Fatalln("ERROR: ", err)
	}
	defer stmt.Close()

	// format all vals at once
	_, err = stmt.Exec(vals...)
	if err != nil {
		log.Error(err)
	}

	return nil

}

// BatchSetStatus updates the status of a batch of upload files.
func (s *manifestFileStore) BatchSetStatus(status manifestFile.Status, uploadIds []string) error {

	UploadIdStr := "('" + strings.Join(uploadIds, "','") + "')"

	query := fmt.Sprintf("UPDATE manifest_files SET status='%s' WHERE upload_id IN %s;", status.String(), UploadIdStr)
	_, err := s.db.Exec(query)
	if err != nil {
		fmt.Sprintln("Unable to update manifest file status for batch. Here is why: ", err)
		return err
	}

	return nil
}

// SetStatus updates status in sqllite config for file
func (s *manifestFileStore) SetStatus(status manifestFile.Status, uploadId string) error {

	statement, err := s.db.Prepare(
		"UPDATE manifest_files SET status=? WHERE upload_id=?")
	if err != nil {
		fmt.Println(err)
		return err
	}

	_, err = statement.Exec(status.String(), uploadId)
	if err != nil {
		fmt.Sprintln("Unable to update manifest file status. Here is why: ", err)
		return err
	}

	return nil
}

// SyncResponseStatusUpdate updates files in a manifest to the provided status for each file.
func (s *manifestFileStore) SyncResponseStatusUpdate(manifestId int32, statusList []manifestFile.FileStatusDTO) error {

	allStatus := []manifestFile.Status{
		manifestFile.Local,
		manifestFile.Registered,
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
	for key, status := range idByStatus {
		if len(status) > 0 {

			// Batch update statements
			// Split the slice into batches of 200 items.
			batch := 250
			for i := 0; i < len(status); i += batch {
				j := i + batch
				if j > len(status) {
					j = len(status)
				}

				allUploadIds := strings.Join(status[i:j], ",")
				sqlStatement := fmt.Sprintf("UPDATE manifest_files SET status = '%s' "+
					"WHERE manifest_id = %d AND upload_id IN (%s);", key, manifestId, allUploadIds)

				log.Info("Updating Database with %d rows\n", len(status[i:j]))
				_, err := s.db.Exec(sqlStatement)
				if err != nil {
					log.Error("Unable to update status in manifest files for manifest:", manifestId, "--", err)
					return err
				}
			}
		}
	}

	return nil
}

// SyncResponseStatusUpdate2 updates local DB based on successful/unsuccessful updates remotely.
// 1. Set to SYNCED for all files that were successfully synchronized (Initiated, Failed)
// 2. Remove files with REMOVED that were successfully removed remotely.
func (s *manifestFileStore) SyncResponseStatusUpdate2(manifestId int32, failedFiles []string) {

	// Set INITIATED and FAILED to SYNCED
	requestStatus := []manifestFile.Status{
		manifestFile.Local,
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
		manifestFile.Registered.String(), manifestId, statusQueryString)

	if len(failedList) > 0 {
		failedFilesString := fmt.Sprintf("(%s)", strings.Join(failedList, ","))
		queryString = queryString + fmt.Sprintf(" AND NOT IN %s", failedFilesString)

	}
	queryString = queryString + ";"

	stmt, err := s.db.Prepare(queryString)
	if err != nil {
		log.Fatalln("ERROR: ", err)
	}
	//defer stmt.Close()

	log.Debug(stmt)

	// format all vals at once
	_, err = stmt.Exec()
	if err != nil {
		log.Error(err)
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

	stmt, err = s.db.Prepare(queryString)
	if err != nil {
		log.Fatalln("ERROR: ", err)
	}

	log.Debug(stmt)

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
func (s *manifestFileStore) RemoveFromManifest(manifestId int32, removePath string) error {

	pathLikeExpr := fmt.Sprintf("'%s%%'", removePath)
	initatedStatus := manifestFile.Local.String()
	queryStr := fmt.Sprintf("DELETE FROM manifest_files WHERE manifest_id = %d "+
		"AND source_path LIKE %s and status = '%s';", manifestId, pathLikeExpr, initatedStatus)

	log.Debug(queryStr)

	_, err := s.db.Exec(queryStr)
	if err != nil {
		return err
	}

	syncStatus := manifestFile.Registered.String()
	removeStatus := manifestFile.Removed.String()
	queryStr2 := fmt.Sprintf("UPDATE manifest_files SET status = '%s' WHERE manifest_id = %d "+
		"AND source_path LIKE %s and status = '%s';", removeStatus, manifestId, pathLikeExpr, syncStatus)

	_, err = s.db.Exec(queryStr2)
	if err != nil {
		return err
	}

	return nil
}

// ResetStatusForManifest resets all files to status = LOCAL
func (s *manifestFileStore) ResetStatusForManifest(manifestId int32) error {

	currentTime := time.Now()

	initiatedStatusStr := manifestFile.Local.String()
	sqlStatement := fmt.Sprintf("UPDATE manifest_files SET status = '%s', updated_at = %d WHERE manifest_id = %d",
		initiatedStatusStr, currentTime.Unix(), manifestId)

	log.Debug(sqlStatement)
	// format all vals at once
	_, err := s.db.Exec(sqlStatement)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// GetNumberOfRowsForStatus returns the number of rows in a manifest that do (not) have a specific status
func (s *manifestFileStore) GetNumberOfRowsForStatus(manifestId int32, statusArr []manifestFile.Status, invert bool) (int64, error) {

	var statusStrArr []string
	for _, st := range statusArr {
		statusStrArr = append(statusStrArr, st.String())
	}
	statusStr := fmt.Sprintf("'%s'", strings.Join(statusStrArr, "','"))

	invertedStr := ""
	if invert {
		invertedStr = "NOT"
	}

	// Get Total items to sync
	queryStr := fmt.Sprintf("SELECT  count(*) FROM manifest_files WHERE manifest_id=? AND status %s IN (%s)", invertedStr, statusStr)

	var totalNrRows int64
	err := s.db.QueryRow(queryStr, manifestId).Scan(&totalNrRows)
	switch {
	case err == sql.ErrNoRows:
		return 0, errors.New("unable to get number of rows to be synchronized")
	case err != nil:
		return 0, errors.New("unable to get number of rows to be synchronized")
	default:
		log.Info("About to synchronize %d files.", totalNrRows)
	}

	return totalNrRows, nil
}

// ManifestFilesToChannel streams files in a manifest with a specific status to a channel
func (s *manifestFileStore) ManifestFilesToChannel(ctx context.Context, manifestId int32, statusArr []manifestFile.Status, walker chan<- ManifestFile) {
	// 1. Synchronize Walker

	var statusList []string
	for _, reqStatus := range statusArr {
		statusList = append(statusList, fmt.Sprintf("'%s'", reqStatus.String()))
	}
	statusQueryString := fmt.Sprintf("(%s)", strings.Join(statusList, ","))
	queryStr := fmt.Sprintf("SELECT * FROM manifest_files WHERE manifest_id = ? "+
		"AND status IN %s ORDER BY id", statusQueryString)

	rows, err := s.db.QueryContext(ctx, queryStr, manifestId)
	if err != nil {
		log.Fatal(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Error("Unable to close rows in Upload.")
		}
	}(rows)

	// Iterate over rows for manifest and add row to channel to be picked up by worker.
	for rows.Next() {
		var status string
		currentRecord := ManifestFile{}
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

		if err != nil {
			log.Fatal(err)
		}

		var s manifestFile.Status
		currentRecord.Status = s.ManifestFileStatusMap(status)

		walker <- currentRecord
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}
