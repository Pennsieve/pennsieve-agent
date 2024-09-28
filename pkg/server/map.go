package server

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"time"
)

// Fetch gets a representation of the dataset on the local machine
// NOTE: This does NOT support packages with multiple source-files.
func (s *server) Map(ctx context.Context, req *api.MapRequest) (*api.SimpleStatusResponse, error) {

	var err error

	if _, err := os.Stat(req.TargetFolder); err == nil {
		return &api.SimpleStatusResponse{Status: "Cannot map into folder as folder already exists."}, nil
	}

	client := s.client
	manifestResponse, err := client.Dataset.GetManifest(ctx, req.DatasetId)
	if err != nil {
		log.Errorf("Download failed: %v", err)
		return nil, err
	}

	// Create folder (and include hidden .pennsieve folder for manifest)
	err = os.MkdirAll(path.Join(req.TargetFolder, ".pennsieve"), os.ModePerm)
	if err != nil {
		log.Errorf("Failed to create target path: %v", err)
		return nil, err
	}

	// Download Manifest to hidden .pennsieve folder in targetpath
	manifestLocation := path.Join(req.TargetFolder, ".pennsieve", "manifest.json")
	_, err = s.downloadFileFromPresignedUrl(ctx, manifestResponse.URL, manifestLocation, uuid.New().String())
	if err != nil {
		log.Errorf("Download failed: %v", err)
		return nil, err
	}

	data, err := shared.ReadWorkspaceManifest(manifestLocation)
	if err != nil {
		log.Errorf("Failed to read manifest: %v", err)
		return nil, err
	}

	// Create the state file
	state := models.MapState{
		LastFetch: time.Now(),
		LastPull:  time.Now(),
		Files:     nil,
	}

	stateJson, _ := json.MarshalIndent(state, "", "  ")
	stateFileLocation := path.Join(req.TargetFolder, ".pennsieve", "state.json")
	err = os.WriteFile(stateFileLocation, stateJson, 0644)
	if err != nil {
		return nil, err
	}

	for _, file := range data.Files {

		fileLocation := path.Join(req.TargetFolder, file.Path, file.PackageName)

		err := os.MkdirAll(path.Dir(fileLocation), os.ModePerm)
		if err != nil {
			log.Errorf("Failed to create target path: %v", err)
			continue
		}

		err = touchFile(fileLocation, file.FileNodeId.String)
		if err != nil {
			log.Errorf("Failed to create target file: %v", err)
			continue
		}

	}

	resp := &api.SimpleStatusResponse{Status: "Success"}

	return resp, nil
}

// touchFile creates file and writes the fileID to the file.
// This is used as the 'empty' representation of a file on Pennsieve.
func touchFile(name string, fileUUID string) error {

	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write([]byte(fileUUID)); err != nil {
		f.Close() // ignore error; Write error takes precedence
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	return nil
}
