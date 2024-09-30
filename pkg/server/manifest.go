// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"errors"
	"fmt"
	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/manifest"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/manifest/manifestFile"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type fileWalk chan string

// API ENDPOINT IMPLEMENTATIONS
// --------------------------------------------

// ListManifests returns a list of manifests that are currently defined in the local database.
func (s *server) ListManifests(ctx context.Context, request *pb.ListManifestsRequest) (*pb.ListManifestsResponse, error) {

	manifests, err := s.Manifest.GetAll()

	var r []*pb.ListManifestsResponse_Manifest
	for _, m := range manifests {

		nodeId := ""
		if m.NodeId.Valid {
			nodeId = m.NodeId.String
		}

		r = append(r, &pb.ListManifestsResponse_Manifest{
			Id:               m.Id,
			NodeId:           nodeId,
			UserName:         m.UserName,
			UserId:           m.UserId,
			OrganizationName: m.OrganizationName,
			OrganizationId:   m.OrganizationId,
			DatasetName:      m.DatasetName,
			DatasetId:        m.DatasetId,
			Status:           m.Status.String(),
		})
	}
	response := pb.ListManifestsResponse{Manifests: r}
	return &response, err
}

// CreateManifest recursively adds paths from folder into local DB.
func (s *server) CreateManifest(ctx context.Context, request *pb.CreateManifestRequest) (*pb.CreateManifestResponse, error) {

	// 1. Get new Upload Session ID from Pennsieve Server
	// --------------------------------------------------
	activeUser, err := s.User.GetActiveUser()
	if err != nil {
		log.Error("Cannot get active user: ", err)
	}

	curClientSession, err := s.User.GetUserSettings()
	if err != nil {
		err := status.Error(codes.NotFound,
			"Unable to get Client Session\n "+
				"\t Please use: pennsieve config init to initialize local database.")

		log.Warn(err)
		return nil, err
	}

	// Check that there is an active dataset
	if curClientSession.UseDatasetId == "" {
		err := status.Error(codes.NotFound,
			"No active dataset was specified.\n "+
				"\t Please use: pennsieve dataset use <dataset_id> to specify active dataset.")

		log.Warn(err)
		return nil, err
	}

	// Check dataset exist (should be redundant) and grab name
	ds, err := s.client.Dataset.Get(nil, curClientSession.UseDatasetId)
	if err != nil {
		log.Error(err)
	}

	newSession := store.ManifestParams{
		UserId:           activeUser.Id,
		UserName:         activeUser.Name,
		OrganizationId:   activeUser.OrganizationId,
		OrganizationName: activeUser.OrganizationName,
		DatasetId:        curClientSession.UseDatasetId,
		DatasetName:      ds.Content.Name,
	}

	createdManifest, err := s.Manifest.Add(newSession)
	if err != nil {
		err := status.Error(codes.NotFound,
			"Unable to create Upload Session.\n "+
				"\t Please use: pennsieve config init to initialize local database.")

		log.Warn(err)
		return nil, err
	}

	// 2. Walk over folder and populate DB with file-paths.
	// --------------------------------------------------
	nrRecords, err := s.addToManifest(request.BasePath, request.TargetBasePath, request.Files, createdManifest.Id)
	if err != nil {
		s.messageSubscribers(fmt.Sprintf("Error with accessing files on: %s", request.BasePath))
		response := pb.CreateManifestResponse{ManifestId: createdManifest.Id, Message: fmt.Sprintf("Creating manifest failed.")}
		return &response, nil
	}

	s.messageSubscribers(fmt.Sprintf("Finished Adding %d files to Manifest.", nrRecords))

	response := pb.CreateManifestResponse{ManifestId: createdManifest.Id, Message: fmt.Sprintf("Successfully indexed %d files.", nrRecords)}
	return &response, nil

}

// AddToManifest adds files to existing upload manifest.
func (s *server) AddToManifest(ctx context.Context, request *pb.AddToManifestRequest) (*pb.SimpleStatusResponse, error) {

	nrRecords, _ := s.addToManifest(request.BasePath, request.TargetBasePath, request.Files, request.ManifestId)

	log.Info(fmt.Sprintf("Finished Adding %d files.", nrRecords))

	response := pb.SimpleStatusResponse{Status: fmt.Sprintf("Successfully indexed %d files.", nrRecords)}
	return &response, nil
}

// RemoveFromManifest removes one or more files from the index for an existing manifest.
func (s *server) RemoveFromManifest(ctx context.Context, request *pb.RemoveFromManifestRequest) (*pb.SimpleStatusResponse, error) {

	err := s.Manifest.RemoveFromManifest(request.ManifestId, request.RemovePath)
	if err != nil {
		return nil, err
	}

	response := pb.SimpleStatusResponse{Status: fmt.Sprintf("Successfully removed files.")}
	return &response, nil
}

// DeleteManifest deletes existing upload manifest.
func (s *server) DeleteManifest(ctx context.Context, request *pb.DeleteManifestRequest) (*pb.SimpleStatusResponse, error) {

	//	1. Verify that manifest with ID exists

	//	2. TODO: Remove/cancel manifest from server

	//	3. Delete manifest from local database

	err := s.Manifest.RemoveManifest(request.ManifestId)

	if err != nil {
		err := status.Error(codes.NotFound,
			"Unable to remove upload manifest\n "+
				"\t Check if manifest exists..")

		log.Error(err)
		return nil, err
	}

	response := pb.SimpleStatusResponse{Status: "Success"}
	return &response, nil

}

// ListManifestFiles lists files from an existing upload manifest.
func (s *server) ListManifestFiles(ctx context.Context, request *pb.ListManifestFilesRequest) (*pb.ListManifestFilesResponse, error) {

	result, err := s.Manifest.GetFiles(request.ManifestId, request.Limit, request.Offset)
	if err != nil {
		return nil, err
	}

	var r []*pb.ListManifestFilesResponse_FileUpload
	for _, m := range result {

		statusInt := pb.ListManifestFilesResponse_StatusType_value[strings.ToUpper(m.Status.String())]
		st := pb.ListManifestFilesResponse_StatusType(statusInt)

		r = append(r, &pb.ListManifestFilesResponse_FileUpload{
			Id:         m.Id,
			ManifestId: m.ManifestId,
			SourcePath: m.SourcePath,
			TargetPath: m.TargetPath,
			UploadId:   m.UploadId.String(),
			Status:     st,
		})

	}

	response := pb.ListManifestFilesResponse{File: r}

	return &response, nil

}

// SyncManifest synchronizes the state of the manifest between local and cloud server.
func (s *server) SyncManifest(ctx context.Context, request *pb.SyncManifestRequest) (*pb.SyncManifestResponse, error) {

	/*
		ManifestSync only synchronizes manifest files of status:
		- FileInitated
		- FileFailed
		- FileRemoved

		If successful, files with those statuses will be updated in the local config where
		Initiate, Failed --> Synced
		Removed --> (file removed from local config)
	*/

	manifest, err := s.Manifest.GetManifest(request.ManifestId)
	if err != nil {
		return nil, err
	}

	// Verify uploaded files that are in Finalized state.
	if manifest.NodeId.Valid {
		log.Debug("Verifying files")
		s.Manifest.VerifyFinalizedStatus(manifest)
	}

	// Sync local files with the server.
	log.Debug("Syncing files.")

	// Create Context and store cancel function in server object.
	ctx, cancelFnc := context.WithCancel(context.Background())
	session := uploadSession{
		manifestId: request.GetManifestId(),
		cancelFnc:  cancelFnc,
	}
	s.cancelFncs.Store(request.GetManifestId(), session)

	go s.syncProcessor(ctx, manifest)

	r := pb.SyncManifestResponse{
		ManifestNodeId: manifest.NodeId.String,
	}

	return &r, nil

}

// RelocateManifestFiles allows users to update the target path for a given path.
func (s *server) RelocateManifestFiles(ctx context.Context, request *pb.RelocateManifestFilesRequest) (*pb.SimpleStatusResponse, error) {

	return nil, nil
}

// ResetManifest allows users to reset the status for all files in a manifest
func (s *server) ResetManifest(ctx context.Context, request *pb.ResetManifestRequest) (*pb.SimpleStatusResponse, error) {

	err := s.Manifest.ResetStatusForManifest(request.ManifestId)
	if err != nil {
		log.Error("Cannot reset manifest: ", err)
		return nil, err
	}

	response := pb.SimpleStatusResponse{Status: "Success"}
	return &response, nil
}

// ----------------------------------------------
// SYNC FUNCTIONS
// ----------------------------------------------

type syncSummary struct {
	nrFilesUpdated int
}

// syncProcessor Go routine that manages sync go sub-routines for crawling DB and syncing rows with service
func (s *server) syncProcessor(ctx context.Context, m *store.Manifest) (*syncSummary, error) {

	log.Debug("IN SYNC PROCESSOR")

	nrWorkers := viper.GetInt("agent.upload_workers")
	syncWalker := make(chan store.ManifestFile, nrWorkers)
	syncResults := make(syncResult, nrWorkers)

	totalNrRows, err := s.Manifest.GetNumberOfRowsForStatus(m.Id,
		[]manifestFile.Status{manifestFile.Verified, manifestFile.Uploaded, manifestFile.Registered}, true)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Database crawler to fetch rows
	go func() {
		defer close(syncWalker)

		requestStatus := []manifestFile.Status{
			manifestFile.Local,
			manifestFile.Changed,
			manifestFile.Failed,
			manifestFile.Removed,
			manifestFile.Imported,
			manifestFile.Finalized,
			manifestFile.Unknown,
		}

		s.Manifest.ManifestFilesToChannel(ctx, m.Id, requestStatus, syncWalker)

	}()

	// Sync handler
	go func() {
		defer close(syncResults)

		var syncWaitGroup sync.WaitGroup

		// 1. Ensure we get a manifest id from the server
		err := s.getCreateManifestId(m)
		if err != nil {
			log.Error("Error: ", err)
			return
		}

		s.syncUpdateSubscribers(totalNrRows, 0, 0, pb.SubscribeResponse_SyncResponse_INIT)

		// 2. Create Sync Workers
		for w := 1; w <= nrWorkers; w++ {
			syncWaitGroup.Add(1)

			log.Debug("starting Sync Worker:", w)
			workerId := int32(w)
			go func() {
				defer func() {
					syncWaitGroup.Done()
					log.Debug("stopping Sync Worker:", w)
				}()
				err := s.syncWorker(ctx, workerId, syncWalker, syncResults, m, totalNrRows)
				if err != nil {
					log.Error("Error in Upload Worker:", err)
				}
			}()
		}

		syncWaitGroup.Wait()
		s.syncUpdateSubscribers(totalNrRows, 0, 0, pb.SubscribeResponse_SyncResponse_COMPLETE)
		log.Info("All sync workers complete --> closing syncResults channel")
	}()

	// Handling results from sync handlers.
	// This for loop with continuously add responses from service to array of updated files.
	// It stops when syncResults is closed, which happens when all sync handlers have finished.
	var allStatusUpdates []manifestFile.FileStatusDTO
	for result := range syncResults {
		allStatusUpdates = append(allStatusUpdates, result...)
	}

	// Update file status for synchronized manifest.
	log.Info("Updating local database with status results.")
	s.Manifest.SyncResponseStatusUpdate(m.Id, allStatusUpdates)

	return &syncSummary{nrFilesUpdated: len(allStatusUpdates)}, nil

}

// getCreateManifestId takes a manifest and ensures the manifest has a node-id.
// The method checks if the manifest has a node-id, and if not, registers the manifest
// with Pennsieve model service and sets the returned node-id in the manifest object.
func (s *server) getCreateManifestId(m *store.Manifest) error {

	// Return if the node id is already set.
	if m.NodeId.Valid {
		return nil
	}

	log.Info("Getting new manifest ID for dataset: ", m.DatasetId)

	requestBody := manifest.DTO{
		DatasetId: m.DatasetId,
	}

	client := s.client

	response, err := client.Manifest.Create(context.Background(), requestBody)
	if err != nil {
		log.Error("ERROR: Unable to get new manifest ID: ", err)
		return err
	}

	log.Debug("New Manifest ID: ", response.ManifestNodeId)
	if response.ManifestNodeId == "" {
		return errors.New("Error: Unexpected Manifest Node ID returned by Pennsieve.")
	}

	// Update NodeId in manifest and database
	s.Manifest.SetManifestNodeId(m, response.ManifestNodeId)

	return nil
}

// syncWorker fetches rows from crawler and syncs with the service by batch.
// This function is called as a go-routine and typically runs multiple instances in parallel
func (s *server) syncWorker(ctx context.Context, workerId int32,
	syncWalker <-chan store.ManifestFile, result chan []manifestFile.FileStatusDTO, m *store.Manifest, totalNrRows int64) error {

	const pageSize = 250

	log.Debug("In SYNC WORKER")

	// Ensure that manifestID is set
	if !m.NodeId.Valid {
		return errors.New("Error: Cannot call syncWorker on manifest that has no manifest node id. ")
	}

	var requestFiles []manifestFile.FileDTO
	for {
		item, ok := <-syncWalker
		if !ok {
			// Final batch of items
			s.syncUpdateSubscribers(totalNrRows, int64(len(requestFiles)), workerId, pb.SubscribeResponse_SyncResponse_IN_PROGRESS)
			log.Debug("Nr Items:", len(requestFiles))
			response, err := s.syncItems(requestFiles, m.NodeId.String, m)
			if err != nil {
				requestFiles = nil
				continue
			}
			result <- response.UpdatedFiles
			requestFiles = nil
			break
		}

		// TODO: CHeck that we can safely remove this as s3-key is no longer used in service
		s3Key := fmt.Sprintf("%s/%s", m.NodeId.String, item.UploadId)

		reqFile := manifestFile.FileDTO{
			UploadID:   item.UploadId.String(),
			S3Key:      s3Key,
			TargetPath: item.TargetPath,
			TargetName: item.TargetName,
			Status:     item.Status,
		}
		requestFiles = append(requestFiles, reqFile)

		if len(requestFiles) == pageSize {
			s.syncUpdateSubscribers(totalNrRows, pageSize, workerId, pb.SubscribeResponse_SyncResponse_IN_PROGRESS)
			response, err := s.syncItems(requestFiles, m.NodeId.String, m)
			if err != nil {
				requestFiles = nil
				continue
			}
			result <- response.UpdatedFiles

			requestFiles = nil
		}

	}
	return nil
}

func (s *server) syncItems(requestFiles []manifestFile.FileDTO, manifestNodeId string, m *store.Manifest) (*manifest.PostResponse, error) {

	requestBody := manifest.DTO{
		DatasetId: m.DatasetId,
		ID:        manifestNodeId,
		Files:     requestFiles,
		Status:    m.Status,
	}

	response, err := s.client.Manifest.Create(context.Background(), requestBody)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return response, nil
}

// HELPER FUNCTIONS
// ----------------------------------------------

// updateSubscribers sends upload-progress updates to all grpc-update subscribers.
func (s *server) syncUpdateSubscribers(total int64, nrSynced int64, workerId int32, status pb.SubscribeResponse_SyncResponse_SyncStatus) {
	// A list of clients to unsubscribe in case of error
	var unsubscribe []int32

	// Iterate over all subscribers and send data to each client
	s.subscribers.Range(func(k, v interface{}) bool {
		id, ok := k.(int32)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber key: %T", k))
			return false
		}
		sub, ok := v.(sub)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber value: %T", v))
			return false
		}
		// Send data over the gRPC stream to the client
		if err := sub.stream.Send(&pb.SubscribeResponse{
			Type: pb.SubscribeResponse_SYNC_STATUS,
			MessageData: &pb.SubscribeResponse_SyncStatus{
				SyncStatus: &pb.SubscribeResponse_SyncResponse{
					Total:    total,
					Status:   status,
					NrSynced: nrSynced,
					WorkerId: workerId,
				}},
		}); err != nil {
			log.Error(fmt.Sprintf("Failed to send data to client: %v", err))
			select {
			case sub.finished <- true:
				log.Info(fmt.Sprintf("Unsubscribed client: %d", id))
			default:
				// Default case is to avoid blocking in case client has already unsubscribed
			}
			// In case of error the client would re-subscribe so close the subscriber stream
			unsubscribe = append(unsubscribe, id)
		}
		return true
	})

	// Unsubscribe erroneous client streams
	for _, id := range unsubscribe {
		s.subscribers.Delete(id)
	}
}

func (f fileWalk) Walk(path string, info fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !info.IsDir() {
		f <- path
	}
	return nil
}

// addToManifest walks over provided path and adds records to DB
func (s *server) addToManifest(localBasePath string, targetBasePath string, files []string, manifestId int32) (int, error) {

	if len(files) > 0 && len(localBasePath) > 0 {
		err := status.Error(codes.NotFound,
			"Unable to add to Manifest.\n "+
				"\t You cannot specify both 'basePath' and 'files'.")

		log.Error(err)
		return 0, err

	}

	batchSize := 50 // Update DB with 50 paths per batch
	walker := make(fileWalk, batchSize)
	errs := make(chan error, 1) // Use channel to export error if walk fails.
	go func() {

		if len(files) > 0 {
			for _, f := range files {
				walker <- f
			}
		} else {
			// Gather the files to upload by walking the path recursively
			if err := filepath.WalkDir(localBasePath, walker.Walk); err != nil {
				log.Error("Walk failed:", err)
				errs <- fmt.Errorf("walkError: Unable to read: %s", localBasePath)
			}
		}

		close(walker)
		close(errs)
	}()

	// Get paths from channel, and when <batchSize> number of paths,
	// store these in the local DB.
	totalIndexed := 0
	i := 0
	var items []string
	for {
		item, ok := <-walker
		if !ok {
			// Final batch of items
			err := s.addUploadRecords(items, localBasePath, targetBasePath, manifestId)
			if err != nil {
				return 0, err
			}
			totalIndexed += len(items)
			break
		}

		items = append(items, item)
		i++
		if i == batchSize {
			// Standard batch of items
			err := s.addUploadRecords(items, localBasePath, targetBasePath, manifestId)
			if err != nil {
				return 0, err
			}

			i = 0
			totalIndexed += batchSize
			items = nil
		}
	}

	hasError := <-errs

	return totalIndexed, hasError
}

// addUploadRecords adds records to the local SQLite DB.
func (s *server) addUploadRecords(paths []string, localBasePath string, targetBasePath string, manifestId int32) error {

	records := recordsFromPaths(paths, localBasePath, targetBasePath, manifestId)

	if len(records) > 0 {
		err := s.Manifest.AddFiles(records)
		if err != nil {
			log.Error("Error with AddUploadRecords: ", err)
			return err
		}
	}

	return nil
}

// recordsFromPaths creates a set of records to be stored in the dynamodb from a list of paths.
func recordsFromPaths(paths []string, localBasePath string, targetBasePath string, manifestId int32) []store.ManifestFileParams {
	var records []store.ManifestFileParams
	for _, row := range paths {
		if len(localBasePath) > 0 && shared.PathIsDirectory(localBasePath) {
			// localBasePath was provided, and it is a folder/directory
			fileName, targetPath := fileTargetPath(row, localBasePath, targetBasePath)
			newRecord := store.ManifestFileParams{
				SourcePath: row,
				TargetPath: targetPath,
				TargetName: fileName,
				ManifestId: manifestId,
			}
			records = append(records, newRecord)
		} else {
			// localBasePath was not provided, or it is the path to a file
			fileName := filepath.Base(row)
			targetPath := targetBasePath
			newRecord := store.ManifestFileParams{
				SourcePath: row,
				TargetPath: targetPath,
				TargetName: fileName,
				ManifestId: manifestId,
			}
			records = append(records, newRecord)
		}
	}

	return records
}

func fileTargetPath(file string, basePath string, targetBasePath string) (string, string) {
	relPath, err := filepath.Rel(basePath, file)
	if err != nil {
		log.Fatal("Cannot strip base-path.")
	}

	// ensure path separator is slash
	relPath = filepath.ToSlash(relPath)

	r2 := regexp.MustCompile(`(?P<Path>([^\/]*\/)*)(?P<FileName>[^\.]*)?\.?(?P<Extension>.*)`)
	pathParts := r2.FindStringSubmatch(relPath)

	filePath := pathParts[r2.SubexpIndex("Path")]
	fileExtension := pathParts[r2.SubexpIndex("Extension")]
	str := []string{pathParts[r2.SubexpIndex("FileName")], fileExtension}
	fileName := strings.Join(str, ".")

	targetPath := filepath.Join(targetBasePath, filePath)
	targetPath = filepath.ToSlash(targetPath)

	return fileName, targetPath
}
