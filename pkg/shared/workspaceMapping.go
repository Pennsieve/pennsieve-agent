package shared

import (
	"encoding/json"
	"errors"
	"fmt"
	models2 "github.com/pennsieve/pennsieve-agent/pkg/models"
	models "github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

func ReadStateFile(stateFileLocation string) (*models2.MapState, error) {

	// Now read in manifest
	manifestFile, err := os.Open(stateFileLocation)
	if err != nil {
		fmt.Printf("failed to open manifest file: %s, error: %v", stateFileLocation, err)
		return nil, err
	}
	defer func(manifestFile *os.File) {
		err := manifestFile.Close()
		if err != nil {
			log.Warn("Unable to close manifest file")
		}
	}(manifestFile)

	jsonData, err := io.ReadAll(manifestFile)
	if err != nil {
		fmt.Printf("failed to read json file, error: %v", err)
		return nil, err
	}

	data := models2.MapState{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		fmt.Printf("failed to unmarshal json file, error: %v", err)
		return nil, err
	}

	return &data, nil
}

func ReadWorkspaceManifest(manifestLocation string) (*models.WorkspaceManifest, error) {

	// Now read in manifest
	manifestFile, err := os.Open(manifestLocation)
	if err != nil {
		fmt.Printf("failed to open manifest file: %s, error: %v", manifestLocation, err)
		return nil, err
	}
	defer func(manifestFile *os.File) {
		err := manifestFile.Close()
		if err != nil {
			log.Warn("Unable to close manifest file")
		}
	}(manifestFile)

	jsonData, err := io.ReadAll(manifestFile)
	if err != nil {
		fmt.Printf("failed to read json file, error: %v", err)
		return nil, err
	}

	data := models.WorkspaceManifest{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		fmt.Printf("failed to unmarshal json file, error: %v", err)
		return nil, err
	}

	return &data, nil
}

func ReadFileIDFromFile(location string) (string, error) {

	log.Info(location)
	fi, err := os.Stat(location)
	if err != nil {
		return "", err
	}

	if fi.Size() > 1024 {
		return "", errors.New("file size too large for expected file")
	}

	b, err := os.ReadFile(location)
	if err != nil {
		return "", err
	}

	idStr := string(b)

	return idStr, nil
}
