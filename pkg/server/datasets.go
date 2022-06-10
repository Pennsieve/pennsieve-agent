package server

import (
	"context"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"log"
)

// UseDataset sets the active dataset for the Agent
func (s *server) UseDataset(ctx context.Context, request *pb.UseDatasetRequest) (*pb.UseDatasetResponse, error) {

	// 1. Verify that the dataset exists
	datasetId := request.DatasetId
	client := api.PennsieveClient
	_, err := client.Dataset.Get(nil, datasetId)
	if err != nil {
		log.Fatalln("Unknown dataset: ", datasetId)
	}

	// 2. Update UserSettings to contain dataset ID
	var userSettings models.UserSettings
	err = userSettings.UpdateActiveDataset(datasetId)
	if err != nil {
		log.Fatalln("Unable to update UserSettings:", err)
	}

	response := pb.UseDatasetResponse{DatasetId: datasetId}

	return &response, nil
}
