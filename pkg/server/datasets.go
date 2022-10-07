package server

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/pkg/errors"
	"log"
)

// UseDataset sets the active dataset for the Agent
func (s *server) UseDataset(ctx context.Context, request *pb.UseDatasetRequest) (*pb.UseDatasetResponse, error) {

	// 1. Verify that the dataset exists
	datasetId := request.DatasetId
	client := api.PennsieveClient
	_, err := client.Dataset.Get(nil, datasetId)
	if err != nil {
		log.Printf("UseDataset: Unknown dataset: %s", datasetId)
		return nil, errors.New(fmt.Sprintf("Unknown Dataset: %s", datasetId))
	}

	// 2. Update UserSettings to contain dataset ID
	var userSettings models.UserSettings
	err = userSettings.UpdateActiveDataset(datasetId)
	if err != nil {
		log.Printf("Unable to update UserSettings: %v", err)
		return nil, errors.New("Unable to update local user settings:\n Please re-install the Pennsieve Agent.")
	}

	response := pb.UseDatasetResponse{DatasetId: datasetId}
	return &response, nil
}
