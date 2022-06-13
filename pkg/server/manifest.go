// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

type fileWalk chan string

// API ENDPOINT IMPLEMENTATIONS
// --------------------------------------------

// ListManifests returns a list of manifests that are currently defined in the local database.
func (s *server) ListManifests(ctx context.Context, request *pb.ListManifestsRequest) (*pb.ListManifestsResponse, error) {
	var uploadSession models.Manifest
	manifests, err := uploadSession.GetAll()

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
func (s *server) CreateManifest(ctx context.Context, request *pb.CreateManifestRequest) (*pb.SimpleStatusResponse, error) {

	// 1. Get new Upload Session ID from Pennsieve Server
	// --------------------------------------------------

	//TODO replace with real call to server
	//uploadSessionID := uuid.New()

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

	newSession := models.ManifestParams{
		UserId:           activeUser.Id,
		UserName:         activeUser.Name,
		OrganizationId:   activeUser.OrganizationId,
		OrganizationName: activeUser.OrganizationName,
		DatasetId:        curClientSession.UseDatasetId,
		DatasetName:      ds.Content.Name,
	}

	var manifest models.Manifest
	createdManifest, err := manifest.Add(newSession)
	if err != nil {
		err := status.Error(codes.NotFound,
			"Unable to create Upload Session.\n "+
				"\t Please use: pennsieve-server config init to initialize local database.")

		log.Println(err)
		return nil, err
	}

	// 2. Walk over folder and populate DB with file-paths.
	// --------------------------------------------------
	nrRecords, _ := addToManifest(request.BasePath, request.TargetBasePath, createdManifest.Id)

	log.Printf("Finished Processing %d files.\n", nrRecords)

	response := pb.SimpleStatusResponse{Status: fmt.Sprintf("Successfully indexed %d files.", nrRecords)}
	return &response, nil

}

// AddToManifest adds files to existing upload manifest.
func (s *server) AddToManifest(ctx context.Context, request *pb.AddToManifestRequest) (*pb.SimpleStatusResponse, error) {
	nrRecords, _ := addToManifest(request.BasePath, request.TargetBasePath, request.ManifestId)

	log.Printf("Finished Adding %d files.\n", nrRecords)

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

	var uploadSession models.Manifest
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

// ListManifestFiles lists files from an existing upload manifest.
func (s *server) ListManifestFiles(ctx context.Context, request *pb.ListManifestFilesRequest) (*pb.ListManifestFilesResponse, error) {
	var uploadRecords models.ManifestFile
	result, err := uploadRecords.Get(request.ManifestId, request.Limit, request.Offset)
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

//SyncManifest synchronizes the state of the manifest between local and cloud server.
func (s *server) SyncManifest(ctx context.Context, request *pb.SyncManifestRequest) (*pb.SyncManifestResponse, error) {

	// Manifest Status:
	// (INITIATED, IN_PROGRESS, COMPLETED, CANCELED)

	// Manifest File Status:
	// (LOCAL, PENDING, UPLOADING, COMPLETED, CANCELED)

	// 1. Iterate over files in local manifest
	// - if status is NEW --> upload
	// - if status is DELETED --> remove
	// - if status is SYNCED, UPLOADED --> ignore
	//
	// 2. Call create endpoint without manifest id on first batch with 1000 files
	// 3. Add returned manifestId to Local manifest.
	// 4. Recurrently call endpoint with returned manifest id
	// 5. Same process for removing files.
	//
	// 6. Call Sync endpoint and update local manifest with status updates from server.
	// -

	var m models.Manifest
	manifest, err := m.Get(request.ManifestId)
	if err != nil {
		return nil, err
	}

	manifestNodeId := ""
	if manifest.NodeId.Valid {
		manifestNodeId = manifest.NodeId.String
	}

	client := api.PennsieveClient
	client.BaseURL = pennsieve.BaseURLV2

	var f models.ManifestFile
	files, err := f.Get(request.ManifestId, 100, 0)

	var requestFiles []pennsieve.ManifestRequestFile
	for _, file := range files {
		s3Key := fmt.Sprintf("%s/%d", manifest.NodeId, f.UploadId)
		reqFile := pennsieve.ManifestRequestFile{
			UploadID:   file.UploadId.String(),
			S3Key:      s3Key,
			TargetPath: file.TargetPath,
			TargetName: file.SourcePath,
		}
		requestFiles = append(requestFiles, reqFile)
	}

	requestBody := pennsieve.ManifestRequest{
		DatasetId: manifest.DatasetId,
		ID:        manifestNodeId,
		Files:     requestFiles,
	}

	fmt.Println(requestBody)

	response, err := client.Manifest.Create(context.Background(), requestBody)
	if err != nil {
		return nil, err
	}

	r := pb.SyncManifestResponse{
		ManifestNodeId: response.ManifestNodeId,
		NrFilesAdded:   int32(response.NrFilesAdded),
	}

	return &r, nil

}

// HELPER FUNCTIONS
// ----------------------------------------------

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
func addToManifest(localBasePath string, targetBasePath string, manifestId int32) (int, error) {
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
func addUploadRecords(paths []string, localBasePath string, targetBasePath string, manifestId int32) error {

	var records []models.ManifestFileParams
	for _, row := range paths {
		relPath, err := filepath.Rel(localBasePath, row)
		if err != nil {
			log.Fatal("Cannot strip base-path.")
		}

		newRecord := models.ManifestFileParams{
			SourcePath: row,
			TargetPath: filepath.Join(targetBasePath, relPath),
			ManifestId: manifestId,
		}
		records = append(records, newRecord)
	}

	fmt.Println(records)

	var record models.ManifestFile
	err := record.Add(records)
	if err != nil {
		log.Println("Error with AddUploadRecords: ", err)
		return err
	}

	return nil
}
