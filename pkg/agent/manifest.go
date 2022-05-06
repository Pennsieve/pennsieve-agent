package agent

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"path/filepath"
)

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

// CreateUploadManifest recursively adds paths from folder into local DB.
func (s *server) CreateUploadManifest(ctx context.Context, request *pb.CreateManifestRequest) (*pb.CreateManifestResponse, error) {

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
				"\t Please use: pennsieve-agent config init to initialize local database.")

		log.Println(err)
		return nil, err
	}

	// Check that there is an active dataset
	if curClientSession.UseDatasetId == "" {
		err := status.Error(codes.NotFound,
			"No active dataset was specified.\n "+
				"\t Please use: pennsieve-agent dataset use <dataset_id> to specify active dataset.")

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
				"\t Please use: pennsieve-agent config init to initialize local database.")

		log.Println(err)
		return nil, err
	}

	// 2. Walk over folder and populate DB with file-paths.
	// --------------------------------------------------

	batchSize := 50 // Update DB with 50 paths per batch
	localPath := request.BasePath
	walker := make(fileWalk, batchSize)
	go func() {
		// Gather the files to upload by walking the path recursively
		if err := filepath.WalkDir(localPath, walker.Walk); err != nil {
			log.Println("Walk failed:", err)
		}
		close(walker)
	}()

	// Get paths from channel, and when <batchSize> number of paths,
	// store these in the local DB.
	i := 0
	var items []string
	for {
		item, ok := <-walker
		if !ok {
			// Final batch of items
			api.AddUploadRecords(items, "", uploadSessionID.String())
			break
		}

		items = append(items, item)
		i++
		if i == batchSize {
			// Standard batch of items
			api.AddUploadRecords(items, "", uploadSessionID.String())

			i = 0
			items = nil
		}
	}

	log.Println("Finished Processing files.")

	response := pb.CreateManifestResponse{Status: "Success"}
	return &response, nil

}

// DeleteUploadManifest deletes existing upload manifest.
func (s *server) DeleteUploadManifest(ctx context.Context, request *pb.DeleteManifestRequest) (*pb.DeleteManifestResponse, error) {

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

	response := pb.DeleteManifestResponse{Status: "Success"}
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
