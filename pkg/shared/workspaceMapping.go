package shared

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	models2 "github.com/pennsieve/pennsieve-agent/pkg/models"
	models "github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
	log "github.com/sirupsen/logrus"
)

func ReadStateFile(stateFileLocation string) (*models2.MapState, error) {

	// just in case input is not correct for operating system.
	stateFileLocation = filepath.FromSlash(stateFileLocation)

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
	manifestFile, err := os.Open(filepath.FromSlash(manifestLocation))
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

	f, err := os.Open(filepath.FromSlash(location))
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

// FindManifestFile Looks for .pennsieve/manifest.json starting from the given path
// and moving up through parent directories until found or root is reached
func FindManifestFile(startPath string) (string, error) {
	currentPath := startPath

	for {
		manifestPath := filepath.Join(currentPath, ".pennsieve", "manifest.json")
		if _, err := os.Stat(manifestPath); err == nil {
			return manifestPath, nil
		}

		// Move up to parent directory
		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			// We've reached the root
			return "", fmt.Errorf("manifest.json not found in path hierarchy")
		}
		currentPath = parentPath
	}
}

// WriteWorkspaceManifest writes a workspace manifest to the specified location
func WriteWorkspaceManifest(manifestLocation string, manifest *models.WorkspaceManifest) error {
	manifestFile, err := os.Create(filepath.FromSlash(manifestLocation))
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %s, error: %v", manifestLocation, err)
	}
	defer func(manifestFile *os.File) {
		err := manifestFile.Close()
		if err != nil {
			log.Warn("Unable to close manifest file")
		}
	}(manifestFile)

	// Write manifest with indentation for readability
	encoder := json.NewEncoder(manifestFile)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to encode manifest file: %v", err)
	}

	return nil
}

// CreateManifestDTO creates a ManifestDTO entry for a file
func CreateManifestDTO(fileName string, path string, size int64) models.ManifestDTO {
	return models.ManifestDTO{
		PackageNodeId: "",
		PackageName:   fileName,
		FileNodeId:    models.NullString{},
		FileName: models.NullString{
			NullString: sql.NullString{
				String: fileName,
				Valid:  true,
			},
		},
		Path: path,
		Size: models.NullInt{
			NullInt64: sql.NullInt64{
				Int64: size,
				Valid: true,
			},
		},
		CheckSum: models.NullString{},
	}
}
