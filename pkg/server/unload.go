package server

import (
	"context"
	"encoding/json"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	"os"
	"path/filepath"
)

// Unload provides the following functionality:
// * User downloaded file --> replace by pointer
func (s *server) Unload(ctx context.Context, req *api.UnloadRequest) (*api.SimpleStatusResponse, error) {

	// Check if the provided path is part of a mapped dataset
	datasetRoot, found, err := findMappedDatasetRoot(req.Path)
	if err != nil {
		return nil, err
	}

	if !found {
		return &api.SimpleStatusResponse{Status: "The provided path is not part of a Pennsieve mapped dataset."}, nil
	}

	// Read Dataset State
	state, err := shared.ReadStateFile(filepath.Join(datasetRoot, ".pennsieve", "state.json"))
	if err != nil {
		return nil, err
	}

	// Find packages that are downloaded in the selected path (file/folder) and
	// then replace the file by the reference file to the object.
	for i, f := range state.Files {
		absPath := filepath.Join(datasetRoot, f.Path)
		if f.IsLocal && absPath == req.Path {
			state.Files[i].IsLocal = false
			err = touchFile(absPath, f.FileId)
			if err != nil {
				return nil, err
			}
		}
	}

	// Save the State File
	writeState := models.MapState{
		LastFetch: state.LastFetch,
		LastPull:  state.LastPull,
		Files:     state.Files,
	}

	stateJson, _ := json.MarshalIndent(writeState, "", "  ")
	stateFileLocation := filepath.Join(datasetRoot, ".pennsieve", "state.json")
	err = os.WriteFile(stateFileLocation, stateJson, 0644)
	if err != nil {
		return nil, err
	}

	resp := &api.SimpleStatusResponse{Status: "Success"}

	return resp, nil
}
