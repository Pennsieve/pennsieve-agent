package server

import (
	"context"
	"github.com/google/uuid"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
)

func (s *server) Fetch(ctx context.Context, req *api.FetchRequest) (*api.SimpleStatusResponse, error) {

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
	err = s.downloadFileFromPresignedUrl(ctx, manifestResponse.URL, manifestLocation, uuid.New().String())
	if err != nil {
		log.Errorf("Download failed: %v", err)
		return nil, err
	}

	data, err := shared.ReadWorkspaceManifest(manifestLocation)
	if err != nil {
		log.Errorf("Failed to read manifest: %v", err)
		return nil, err
	}

	// Depending on Fetch vs Download, either download all the files or create
	// empty files instead
	for _, file := range data.Files {

		err := os.MkdirAll(path.Join(req.TargetFolder, file.Path), os.ModePerm)
		if err != nil {
			log.Errorf("Failed to create target path: %v", err)
			continue
		}

		fileLocation := path.Join(req.TargetFolder, file.Path, file.FileName.String)

		err = touchFile(fileLocation)
		if err != nil {
			log.Errorf("Failed to create target file: %v", err)
			continue
		}

	}

	resp := &api.SimpleStatusResponse{Status: "Success"}

	return resp, nil
}

func touchFile(name string) error {
	file, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}
