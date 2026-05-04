// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/manifest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/manifest/manifestFile"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fileWalk chan string

// API ENDPOINT IMPLEMENTATIONS
// --------------------------------------------

// ListManifests returns a list of manifests that are currently defined in the local database.
func (s *agentServer) ListManifests(ctx context.Context, request *pb.ListManifestsRequest) (*pb.ListManifestsResponse, error) {

	manifests, err := s.ManifestService().GetAll()

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
func (s *agentServer) CreateManifest(ctx context.Context, request *pb.CreateManifestRequest) (*pb.CreateManifestResponse, error) {

	// 1. Get new Upload Session ID from Pennsieve Server
	// --------------------------------------------------
	activeUser, err := s.UserService().GetActiveUser()
	if err != nil {
		log.Error("Cannot get active user: ", err)
		return nil, err
	}

	curClientSession, err := s.UserService().GetUserSettings()
	if err != nil {
		err := status.Error(codes.NotFound,
			"Unable to get Client Session\n "+
				"\t Please use: Pennsieve config init to initialize local database.")

		log.Warn(err)
		return nil, err
	}

	// Check that there is an active dataset
	if curClientSession.UseDatasetId == "" {
		err := status.Error(codes.NotFound,
			"No active dataset was specified.\n "+
				"\t Please use: Pennsieve dataset use <dataset_id> to specify active dataset.")

		log.Warn(err)
		return nil, err
	}

	// Check dataset exist (should be redundant) and grab name
	client, err := s.PennsieveClient()
	if err != nil {
		log.Error("Unable to initialize Pennsieve client: ", err)
		return nil, err
	}

	ds, err := client.Dataset.Get(ctx, curClientSession.UseDatasetId)
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

	createdManifest, err := s.ManifestService().Add(newSession)
	if err != nil {
		err := status.Error(codes.NotFound,
			"Unable to create Upload Session.\n "+
				"\t Please use: pennsieve config init to initialize local database.")

		log.Warn(err)
		return nil, err
	}

	// 2. Walk over folder and populate DB with file-paths.
	// --------------------------------------------------
	nrRecords, nrOSSkipped, nrSecretsSkipped, err := s.addToManifest(request.BasePath, request.TargetBasePath, request.Files, createdManifest.Id)
	if err != nil {
		s.messageSubscribers(fmt.Sprintf("Error with accessing files on: %s", request.BasePath))
		response := pb.CreateManifestResponse{ManifestId: createdManifest.Id, Message: "Creating manifest failed."}
		return &response, nil
	}

	indexedMsg := manifestAddSummary(nrRecords, nrOSSkipped, nrSecretsSkipped)
	s.messageSubscribers("Finished Adding " + indexedMsg)

	response := pb.CreateManifestResponse{ManifestId: createdManifest.Id, Message: "Successfully indexed " + indexedMsg}
	return &response, nil

}

// AddToManifest adds files to existing upload manifest.
func (s *agentServer) AddToManifest(ctx context.Context, request *pb.AddToManifestRequest) (*pb.SimpleStatusResponse, error) {

	nrRecords, nrOSSkipped, nrSecretsSkipped, _ := s.addToManifest(request.BasePath, request.TargetBasePath, request.Files, request.ManifestId)

	log.Infof("Finished Adding %d files (skipped %d OS metadata, %d suspected credential).", nrRecords, nrOSSkipped, nrSecretsSkipped)

	response := pb.SimpleStatusResponse{Status: "Successfully indexed " + manifestAddSummary(nrRecords, nrOSSkipped, nrSecretsSkipped)}
	return &response, nil
}

// manifestAddSummary formats the user-facing tail of an add-to-manifest
// result, e.g. "13 files. Skipped 4 OS metadata. Skipped 1 suspected
// credential file/dir — see agent.log to audit." Empty skip counts are
// omitted to keep the common case clean.
func manifestAddSummary(indexed, osSkipped, secretsSkipped int) string {
	out := fmt.Sprintf("%d files.", indexed)
	if osSkipped > 0 {
		out += fmt.Sprintf(" Skipped %d OS metadata file(s)/dir(s).", osSkipped)
	}
	if secretsSkipped > 0 {
		out += fmt.Sprintf(" Skipped %d suspected credential file(s)/dir(s) — see agent.log to audit.", secretsSkipped)
	}
	return out
}

// RemoveFromManifest removes one or more files from the index for an existing manifest.
func (s *agentServer) RemoveFromManifest(ctx context.Context, request *pb.RemoveFromManifestRequest) (*pb.SimpleStatusResponse, error) {

	removeResp, err := s.ManifestService().RemoveFromManifest(request.ManifestId, request.RemovePath)
	if err != nil {
		return nil, err
	}

	// using uppercase status strings because that's how the user sees them via the CLI manifest list.
	removeStatus := fmt.Sprintf("Successfully removed %d %s files and %d %s files.",
		removeResp.Deleted, strings.ToUpper(manifestFile.Local.String()),
		removeResp.Updated, strings.ToUpper(manifestFile.Registered.String()))

	response := pb.SimpleStatusResponse{Status: removeStatus}
	return &response, nil
}

// DeleteManifest deletes existing upload manifest.
func (s *agentServer) DeleteManifest(ctx context.Context, request *pb.DeleteManifestRequest) (*pb.SimpleStatusResponse, error) {

	//	1. Verify that manifest with ID exists

	//	2. TODO: Remove/cancel manifest from server

	//	3. Delete manifest from local database

	err := s.ManifestService().RemoveManifest(request.ManifestId)

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
func (s *agentServer) ListManifestFiles(ctx context.Context, request *pb.ListManifestFilesRequest) (*pb.ListManifestFilesResponse, error) {

	result, err := s.ManifestService().GetFiles(request.ManifestId, request.Limit, request.Offset)
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
func (s *agentServer) SyncManifest(ctx context.Context, request *pb.SyncManifestRequest) (*pb.SyncManifestResponse, error) {

	/*
		ManifestSync only synchronizes manifest files of status:
		- FileInitated
		- FileFailed
		- FileRemoved

		If successful, files with those statuses will be updated in the local config where
		Initiate, Failed --> Synced
		Removed --> (file removed from local config)
	*/

	manifest, err := s.ManifestService().GetManifest(request.ManifestId)
	if err != nil {
		return nil, err
	}

	// Verify-finalized reconciliation (server→local Verified) is owned by
	// pkg/reconciler now; it polls every 60s for the daemon's lifetime.

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
func (s *agentServer) RelocateManifestFiles(ctx context.Context, request *pb.RelocateManifestFilesRequest) (*pb.SimpleStatusResponse, error) {

	return nil, nil
}

// ResetManifest allows users to reset the status for all files in a manifest
func (s *agentServer) ResetManifest(ctx context.Context, request *pb.ResetManifestRequest) (*pb.SimpleStatusResponse, error) {

	err := s.ManifestService().ResetStatusForManifest(request.ManifestId)
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

// type recordWalk chan record
type syncResult chan []manifestFile.FileStatusDTO
type syncSummary struct {
	nrFilesUpdated int
}

// syncProcessor Go routine that manages sync go Sub-routines for crawling DB and syncing rows with service
func (s *agentServer) syncProcessor(ctx context.Context, m *store.Manifest) (*syncSummary, error) {

	log.Debug("IN SYNC PROCESSOR")

	nrWorkers := viper.GetInt("agent.upload_workers")
	syncWalker := make(chan store.ManifestFile, nrWorkers)

	syncResults := make(syncResult, nrWorkers)

	totalNrRows, err := s.ManifestService().GetNumberOfRowsForStatus(m.Id,
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

		s.ManifestService().ManifestFilesToChannel(ctx, m.Id, requestStatus, syncWalker)

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

	// Drain results (syncWorker already persisted to SQLite per-batch).
	// We still iterate so the final summary count reflects every sync batch.
	nrUpdated := 0
	for result := range syncResults {
		nrUpdated += len(result)
	}

	return &syncSummary{nrFilesUpdated: nrUpdated}, nil
}

// getCreateManifestId takes a manifest and ensures the manifest has a node-id.
// The method checks if the manifest has a node-id, and if not, registers the manifest
// with Pennsieve model service and sets the returned node-id in the manifest object.
func (s *agentServer) getCreateManifestId(m *store.Manifest) error {

	// Return if the node id is already set.
	if m.NodeId.Valid {
		return nil
	}

	log.Info("Getting new manifest ID for dataset: ", m.DatasetId)

	requestBody := manifest.DTO{
		DatasetId: m.DatasetId,
	}

	client, err := s.PennsieveClient()
	if err != nil {
		return err
	}

	response, err := client.Manifest.Create(context.Background(), requestBody)
	if err != nil {
		log.Error("ERROR: Unable to get new manifest ID: ", err)
		return err
	}

	log.Debug("New Manifest ID: ", response.ManifestNodeId)
	if response.ManifestNodeId == "" {
		return errors.New("error: Unexpected Manifest Node ID returned by Pennsieve")
	}

	// Update NodeId in manifest and database
	s.ManifestService().SetManifestNodeId(m, response.ManifestNodeId)

	return nil
}

// syncWorker fetches rows from crawler and syncs with the service by batch.
// This function is called as a go-routine and typically runs multiple instances in parallel.
//
// Each completed batch is written straight through to the local SQLite store
// (not accumulated) so that upload workers running in parallel can start
// picking up Registered files as soon as the first sync batch returns,
// rather than waiting for the entire manifest to register.
func (s *agentServer) syncWorker(
	_ context.Context,
	workerId int32,
	syncWalker <-chan store.ManifestFile,
	result chan []manifestFile.FileStatusDTO,
	m *store.Manifest,
	totalNrRows int64,
) error {

	const pageSize = 250

	log.Debug("In SYNC WORKER")

	// Ensure that manifestID is set
	if !m.NodeId.Valid {
		return errors.New("error: Cannot call syncWorker on manifest that has no manifest node id")
	}

	flush := func(files []manifestFile.FileDTO) {
		if len(files) == 0 {
			return
		}
		response, err := s.syncItems(files, m.NodeId.String, m)
		if err != nil {
			return
		}
		// Persist status transitions immediately so uploadProcessor's walker
		// can pick up newly Registered files without waiting for the rest of
		// the manifest to sync.
		if err := s.ManifestService().SyncResponseStatusUpdate(m.Id, response.UpdatedFiles); err != nil {
			log.Errorf("SyncResponseStatusUpdate error for batch of %d: %v", len(response.UpdatedFiles), err)
		}
		// Still publish to the result channel so syncProcessor can emit
		// progress updates and compute the final summary.
		result <- response.UpdatedFiles
	}

	var requestFiles []manifestFile.FileDTO
	for {
		item, ok := <-syncWalker
		if !ok {
			// Final batch of items
			s.syncUpdateSubscribers(totalNrRows, int64(len(requestFiles)), workerId, pb.SubscribeResponse_SyncResponse_IN_PROGRESS)
			log.Debug("Nr Items:", len(requestFiles))
			flush(requestFiles)
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
			flush(requestFiles)
			requestFiles = nil
		}

	}
	return nil
}

func (s *agentServer) syncItems(requestFiles []manifestFile.FileDTO, manifestNodeId string, m *store.Manifest) (*manifest.PostResponse, error) {

	requestBody := manifest.DTO{
		DatasetId: m.DatasetId,
		ID:        manifestNodeId,
		Files:     requestFiles,
		Status:    m.Status,
	}

	client, err := s.PennsieveClient()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	response, err := client.Manifest.Create(context.Background(), requestBody)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return response, nil
}

// HELPER FUNCTIONS
// ----------------------------------------------

// updateSubscribers sends upload-progress updates to all grpc-update subscribers.
func (s *agentServer) syncUpdateSubscribers(total int64, nrSynced int64, workerId int32, status pb.SubscribeResponse_SyncResponse_SyncStatus) {
	// A list of clients to unsubscribe in case of error
	var unsubscribe []int32

	// Iterate over all subscribers and send data to each client
	s.subscribers.Range(func(k, v any) bool {
		id, ok := k.(int32)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber key: %T", k))
			return false
		}
		sub, ok := v.(shared.Sub)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber value: %T", v))
			return false
		}
		// Send data over the gRPC Stream to the client
		if err := sub.Stream.Send(&pb.SubscribeResponse{
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
			case sub.Finished <- true:
				log.Info(fmt.Sprintf("Unsubscribed client: %d", id))
			default:
				// Default case is to avoid blocking in case client has already unsubscribed
			}
			// In case of error the client would re-subscribe so close the subscriber Stream
			unsubscribe = append(unsubscribe, id)
		}
		return true
	})

	// Unsubscribe erroneous client streams
	for _, id := range unsubscribe {
		s.subscribers.Delete(id)
	}
}

// addToManifest walks over provided path and adds records to DB. Returns
// the number indexed, the number of OS-metadata entries skipped (.DS_Store,
// __MACOSX, etc.), and the number of suspected secret-bearing entries
// skipped (.env, .ssh/, .aws/, ...) so callers can surface the counts to
// the user. Secret skips are logged at WARN with the full path so the user
// can audit what was filtered.
func (s *agentServer) addToManifest(localBasePath string, targetBasePath string, files []string, manifestId int32) (totalIndexed int, osSkipped int, secretsSkipped int, err error) {

	if len(files) > 0 && len(localBasePath) > 0 {
		err := status.Error(codes.NotFound,
			"Unable to add to Manifest.\n "+
				"\t You cannot specify both 'basePath' and 'files'.")

		log.Error(err)
		return 0, 0, 0, err

	}

	batchSize := 50 // Update DB with 50 paths per batch
	walker := make(fileWalk, batchSize)
	errs := make(chan error, 1) // Use channel to export error if walk fails.
	go func() {

		walkFn := func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := info.Name()
			if info.IsDir() {
				if shared.IsOSNoiseDir(name) {
					osSkipped++
					log.Debugf("addToManifest: skipping OS-noise dir %s", path)
					return filepath.SkipDir
				}
				if shared.IsSecretDir(name) {
					secretsSkipped++
					log.Warnf("addToManifest: skipping suspected credential directory %s (entire subtree excluded)", path)
					return filepath.SkipDir
				}
				return nil
			}
			if shared.IsOSNoiseFile(name) {
				osSkipped++
				log.Debugf("addToManifest: skipping OS-noise file %s", path)
				return nil
			}
			if shared.IsSecretFile(name) {
				secretsSkipped++
				log.Warnf("addToManifest: skipping suspected credential file %s", path)
				return nil
			}
			walker <- path
			return nil
		}

		if len(files) > 0 {
			for _, f := range files {
				walker <- f
			}
		} else {
			// Gather the files to upload by walking the path recursively
			if err := filepath.WalkDir(localBasePath, walkFn); err != nil {
				log.Error("Walk failed:", err)
				errs <- fmt.Errorf("walkError: Unable to read: %s", localBasePath)
			}
		}

		close(walker)
		close(errs)
	}()

	// Get paths from channel, and when <batchSize> number of paths,
	// store these in the local DB.
	i := 0
	var items []string
	for {
		item, ok := <-walker
		if !ok {
			// Final batch of items
			if addErr := s.addUploadRecords(items, localBasePath, targetBasePath, manifestId); addErr != nil {
				return 0, 0, 0, addErr
			}
			totalIndexed += len(items)
			break
		}

		items = append(items, item)
		i++
		if i == batchSize {
			// Standard batch of items
			if addErr := s.addUploadRecords(items, localBasePath, targetBasePath, manifestId); addErr != nil {
				return 0, 0, 0, addErr
			}

			i = 0
			totalIndexed += batchSize
			items = nil
		}
	}

	// Safe to read osSkipped/secretsSkipped here: close(walker) happens after
	// the walker goroutine's last write to the counters, and the consumer
	// loop only exited because it observed walker as closed.
	err = <-errs
	return totalIndexed, osSkipped, secretsSkipped, err
}

// addUploadRecords adds records to the local SQLite DB.
func (s *agentServer) addUploadRecords(paths []string, localBasePath string, targetBasePath string, manifestId int32) error {

	records := recordsFromPaths(paths, localBasePath, targetBasePath, manifestId)

	if len(records) > 0 {
		err := s.ManifestService().AddFiles(records)
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
