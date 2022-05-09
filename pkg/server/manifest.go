// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"path/filepath"
)

// API ENDPOINT IMPLEMENTATIONS
// --------------------------------------------

// ManifestStatus returns a list of manifests that are currently defined in the local database.
func (s *server) ManifestStatus(ctx context.Context, request *pb.ManifestStatusRequest) (*pb.ManifestStatusResponse, error) {
	var uploadSession models.UploadSession
	manifests, err := uploadSession.GetAll()

	var r []*pb.ManifestStatusResponse_Manifest
	for _, m := range manifests {
		r = append(r, &pb.ManifestStatusResponse_Manifest{
			Id:               m.SessionId,
			UserName:         m.UserName,
			UserId:           m.UserId,
			OrganizationName: m.OrganizationName,
			OrganizationId:   m.OrganizationId,
			DatasetName:      m.DatasetName,
			DatasetId:        m.DatasetId,
			Status:           m.Status,
		})
	}
	response := pb.ManifestStatusResponse{Manifests: r}
	return &response, err
}

// CreateManifest recursively adds paths from folder into local DB.
func (s *server) CreateManifest(ctx context.Context, request *pb.CreateManifestRequest) (*pb.SimpleStatusResponse, error) {

	// 1. Get new Upload Session ID from Pennsieve Server
	// --------------------------------------------------

	//TODO replace with real call to server
	uploadSessionID := uuid.New()

	activeUser, _ := api.GetActiveUser()

	var clientSession models.UserSettings
	curClientSession, err := clientSession.Get()
	if err != nil {
		err := status.Error(codes.NotFound,
			"Unable to get Client Session\n "+
				"\t Please use: pennsieve-server config init to initialize local database.")

		log.Println(err)
		return nil, err
	}

	// Check that there is an active dataset
	if curClientSession.UseDatasetId == "" {
		err := status.Error(codes.NotFound,
			"No active dataset was specified.\n "+
				"\t Please use: pennsieve-server dataset use <dataset_id> to specify active dataset.")

		log.Println(err)
		return nil, err
	}

	// Check dataset exist (should be redundant) and grab name
	ds, err := api.PennsieveClient.Dataset.Get(nil, curClientSession.UseDatasetId)

	newSession := models.UploadSessionParams{
		SessionId:        uploadSessionID.String(),
		UserId:           activeUser.Id,
		UserName:         activeUser.Name,
		OrganizationId:   activeUser.OrganizationId,
		OrganizationName: activeUser.OrganizationName,
		DatasetId:        curClientSession.UseDatasetId,
		DatasetName:      ds.Content.Name,
	}

	var uploadSession models.UploadSession
	err = uploadSession.Add(newSession)
	if err != nil {
		err := status.Error(codes.NotFound,
			"Unable to create Upload Session.\n "+
				"\t Please use: pennsieve-server config init to initialize local database.")

		log.Println(err)
		return nil, err
	}

	// 2. Walk over folder and populate DB with file-paths.
	// --------------------------------------------------
	nrRecords, _ := addToManifest(request.BasePath, request.TargetBasePath, uploadSessionID.String())

	log.Println("Finished Processing %d files.", nrRecords)

	response := pb.SimpleStatusResponse{Status: fmt.Sprintf("Successfully indexed %d files.", nrRecords)}
	return &response, nil

}

// AddToManifest adds files to existing upload manifest.
func (s *server) AddToManifest(ctx context.Context, request *pb.AddManifestRequest) (*pb.SimpleStatusResponse, error) {
	nrRecords, _ := addToManifest(request.BasePath, request.TargetBasePath, request.ManifestId)

	log.Println("Finished Adding %d files.", nrRecords)

	response := pb.SimpleStatusResponse{Status: fmt.Sprintf("Successfully indexed %d files.", nrRecords)}
	return &response, nil
}

// RemoveFromManifest removes one or more files from the index for an existing manifest.
func (s *server) RemoveFromManifest(ctx context.Context, request *pb.RemoveFromManifestRequest) (*pb.SimpleStatusResponse, error) {

	response := pb.SimpleStatusResponse{Status: fmt.Sprintf("Successfully indexed %d files.", 0)}
	return &response, nil
}

// DeleteManifest deletes existing upload manifest.
func (s *server) DeleteManifest(ctx context.Context, request *pb.DeleteManifestRequest) (*pb.SimpleStatusResponse, error) {

	//	1. Verify that manifest with ID exists

	//	2. TODO: Remove/cancel manifest from server

	//	3. Delete manifest from local database

	var uploadSession models.UploadSession
	err := uploadSession.Remove(request.ManifestId)

	if err != nil {
		err := status.Error(codes.NotFound,
			"Unable to remove upload manifest\n "+
				"\t Check if manifest exists..")

		log.Println(err)
		return nil, err
	}

	response := pb.SimpleStatusResponse{Status: "Success"}
	return &response, nil

}

// ListFilesForManifest lists files from an existing upload manifest.
func (s *server) ListFilesForManifest(ctx context.Context, request *pb.ListFilesRequest) (*pb.ListFilesResponse, error) {
	var uploadRecords models.UploadRecord
	result, err := uploadRecords.Get(request.ManifestId, request.Limit, request.Offset)
	if err != nil {
		return nil, err
	}

	var r []*pb.ListFilesResponse_FileUpload
	for _, m := range result {

		statusInt := pb.ListFilesResponse_STATUS_TYPE_value[m.Status]
		st := pb.ListFilesResponse_STATUS_TYPE(statusInt)

		r = append(r, &pb.ListFilesResponse_FileUpload{
			Id:         int32(m.Id),
			SessionId:  m.SessionID,
			SourcePath: m.SourcePath,
			TargetPath: m.TargetPath,
			S3Key:      m.S3Key,
			Status:     st,
		})

	}

	response := pb.ListFilesResponse{File: r}

	return &response, nil

}

// HELPER FUNCTIONS
// ----------------------------------------------

// addToManifest walks over provided path and adds records to DB
func addToManifest(localBasePath string, targetBasePath string, manifestId string) (int, error) {
	batchSize := 50 // Update DB with 50 paths per batch
	walker := make(fileWalk, batchSize)
	go func() {
		// Gather the files to upload by walking the path recursively
		if err := filepath.WalkDir(localBasePath, walker.Walk); err != nil {
			log.Println("Walk failed:", err)
		}
		close(walker)
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
			addUploadRecords(items, localBasePath, targetBasePath, manifestId)
			totalIndexed += len(items)
			break
		}

		items = append(items, item)
		i++
		if i == batchSize {
			// Standard batch of items
			addUploadRecords(items, localBasePath, targetBasePath, manifestId)

			i = 0
			totalIndexed += batchSize
			items = nil
		}
	}
	return totalIndexed, nil
}

// addUploadRecords adds records to the local SQLite DB.
func addUploadRecords(paths []string, localBasePath string, targetBasePath string, sessionId string) error {

	var records []models.UploadRecordParams
	for _, row := range paths {
		relPath, err := filepath.Rel(localBasePath, row)
		if err != nil {
			log.Fatal("Cannot strip base-path.")
		}

		newRecord := models.UploadRecordParams{
			SourcePath: row,
			TargetPath: filepath.Join(targetBasePath, relPath),
			S3Key:      uuid.New().String(),
			SessionID:  sessionId,
		}
		records = append(records, newRecord)
	}

	var record models.UploadRecord
	err := record.Add(records)
	if err != nil {
		log.Println("Error with AddUploadRecords: ", err)
		return err
	}

	return nil
}
