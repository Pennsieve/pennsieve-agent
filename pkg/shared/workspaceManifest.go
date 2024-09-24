package shared

import (
	"encoding/json"
	"fmt"
	models "github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

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
