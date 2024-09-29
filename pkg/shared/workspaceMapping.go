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
	"regexp"
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

	f, err := os.Open(location)
	if err != nil {
		return "", err
	}

	defer f.Close()

	var header [36]byte
	_, err = io.ReadFull(f, header[:])
	if err != nil {
		return "", err
	}

	idStr := string(header[:])

	r := regexp.MustCompile(`[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`)
	res := r.FindStringSubmatch(idStr)

	if res == nil {
		return "", errors.New("invalid file id")
	}

	return idStr, nil
}
