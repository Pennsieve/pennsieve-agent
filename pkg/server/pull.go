package server

import (
	"context"
	"encoding/json"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type packageRecord struct {
	PackageId string
	Location  string
}

func (s *server) Pull(ctx context.Context, req *api.PullRequest) (*api.SimpleStatusResponse, error) {

	// Check if the provided path is part of a mapped dataset
	datasetRoot, found, err := findMappedDatasetRoot(req.Path)
	if err != nil {
		return nil, err
	}

	if !found {
		return &api.SimpleStatusResponse{Status: "The provided path is not part of a Pennsieve mapped dataset."}, nil
	}

	// find packageIds associated with the provided path.
	var packages []packageRecord
	workspaceManifest, err := shared.ReadWorkspaceManifest(path.Join(datasetRoot, ".pennsieve", "manifest.json"))
	if err != nil {
		return nil, err
	}

	for _, f := range workspaceManifest.Files {
		if f.FileName.Valid {

			// Check if the file matches or the folder matches.
			// In both cases, add the package
			curFile := path.Join(datasetRoot, f.Path, f.FileName.String)
			curFolder := path.Join(datasetRoot, f.Path)
			if curFile == req.Path || curFolder == req.Path {
				packages = append(packages, packageRecord{
					PackageId: f.PackageNodeId,
					Location:  curFile,
				})
			}
		}
	}

	// Iterate over packages and download files.
	// Run this in a goroutine to prevent blocking of the agent.
	go func() {
		// Open the state file so we can update as needed
		mapState, _ := shared.ReadStateFile(path.Join(datasetRoot, ".pennsieve", "state.json"))

		for _, pkg := range packages {
			client := s.client
			res, err := client.Package.GetPresignedUrl(context.Background(), pkg.PackageId, false)
			if err != nil {
				// TODO: do correct error handling from go routine
				log.Error("Cannot get presigned url")
			}

			// Iterate over the files in a package and download serially
		FILEWALK:
			for _, f := range res.Files {
				_, err := s.downloadFileFromPresignedUrl(ctx, f.URL, pkg.Location, pkg.PackageId)
				if err != nil {
					log.Errorf("Download failed: %v", err)
				}

				// Get CRC for 1st MB of file, or the entire file if less.
				crc32, err := shared.GetFileCrc32(pkg.Location, 1024*1024)
				if err != nil {
					log.Errorf("CRC2 failed: %v", err)
				}

				// Find if entry already exist in state and update if so
				for i, mf := range mapState.Files {
					if mf.Path == pkg.Location {
						mapState.Files[i].PullTime = time.Now()
						mapState.Files[i].Crc32 = crc32
						continue FILEWALK
					}
				}

				// First time we pull the file --> create new record in mapState.
				mapState.Files = append(mapState.Files, models.MapStateRecord{
					Path:     pkg.Location,
					PullTime: time.Now(),
					IsLocal:  true,
					Crc32:    crc32,
				})
			}
		}

		// Update MapState file
		stateJson, _ := json.MarshalIndent(mapState, "", "  ")
		stateFileLocation := path.Join(datasetRoot, ".pennsieve", "state.json")
		err = os.WriteFile(stateFileLocation, stateJson, 0644)

	}()

	resp := &api.SimpleStatusResponse{Status: "Success"}

	return resp, nil
}

// findMappedDatasetRoot checks if the provided path is part of a Pennsieve Mapped Dataset.
func findMappedDatasetRoot(startPath string) (string, bool, error) {

	// Remove extension in case the startPath is a file.
	parentPath := strings.TrimSuffix(startPath, filepath.Ext(startPath))
	manifestPath := ""
	found := false
	var err error

	for parentPath != "/" && parentPath != "." {

		checkLocation := path.Join(parentPath, ".pennsieve", "manifest.json")
		found, err = exists(checkLocation)
		if err != nil {
			return "", found, err
		}
		if found {
			manifestPath = checkLocation
			log.Info(fmt.Sprintf("Found manifest in: %s  ", parentPath))
			break
		}
		parentPath = path.Dir(parentPath)
		log.Info(parentPath)
	}

	if manifestPath == "" {
		log.Info(fmt.Sprintf("%s is not part of a Pennsieve mapped dataset folder.", startPath))

	}

	return parentPath, found, nil

}

// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
