package server

import (
	"context"
	"encoding/json"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/pkg/models"
	"github.com/pennsieve/pennsieve-agent/v2/pkg/shared"
	wsmodels "github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/download"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

func (s *agentServer) Pull(ctx context.Context, req *api.PullRequest) (*api.SimpleStatusResponse, error) {

	// Check if the provided path is part of a mapped dataset
	datasetRoot, found, err := findMappedDatasetRoot(req.Path)
	if err != nil {
		return nil, err
	}

	if !found {
		return &api.SimpleStatusResponse{Status: "The provided path is not part of a Pennsieve mapped dataset."}, nil
	}

	workspaceManifest, err := shared.ReadWorkspaceManifest(filepath.Join(datasetRoot, ".pennsieve", "manifest.json"))
	if err != nil {
		return nil, err
	}
	nodeIds, locations := pullSelection(workspaceManifest, datasetRoot, req.Path)

	client, err := s.PennsieveClient()
	if err != nil {
		return nil, err
	}

	// Download in the background to prevent blocking of the agent;
	// CancelDownload with the dataset id stops it.
	s.startDownload(workspaceManifest.DatasetNodeId, func(ctx context.Context) {
		stateFileLocation := filepath.Join(datasetRoot, ".pennsieve", "state.json")
		mapState, err := shared.ReadStateFile(stateFileLocation)
		if err != nil {
			log.Errorf("Cannot read the map state: %v", err)
			return
		}

		var mu sync.Mutex
		pulled := func(f download.ManifestFile, location string) {
			// Get CRC for 1st MB of file, or the entire file if less.
			crc32, err := shared.GetFileCrc32(location, 1024*1024)
			if err != nil {
				log.Errorf("CRC2 failed: %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			recordPull(mapState, datasetRoot, location, crc32)
		}

		downloader := shared.NewDownloader(s, client)
		for start := 0; start < len(nodeIds) && ctx.Err() == nil; start += shared.MaxManifestNodeIds {
			batch := nodeIds[start:min(start+shared.MaxManifestNodeIds, len(nodeIds))]
			res, err := downloader.DownloadManifest(ctx, shared.ManifestDownload{
				DatasetId: workspaceManifest.DatasetNodeId,
				Request:   download.ManifestRequest{NodeIds: batch},
				Target: func(f download.ManifestFile) string {
					return locations[pullKey{f.NodeId, f.FileName}]
				},
				Done: pulled,
			}, nil)
			logManifestResult(workspaceManifest.DatasetNodeId, res, err)
		}

		// Update MapState file
		stateJson, _ := json.MarshalIndent(mapState, "", "  ")
		if err := os.WriteFile(stateFileLocation, stateJson, 0644); err != nil {
			log.Errorf("Cannot update the map state: %v", err)
		}
	})

	resp := &api.SimpleStatusResponse{Status: "Success"}

	return resp, nil
}

type pullKey struct {
	packageNodeId string
	fileName      string
}

// pullSelection finds the packages to pull for path, a file or folder in the
// mapped dataset at datasetRoot, and where each of their files goes.
func pullSelection(m *wsmodels.WorkspaceManifest, datasetRoot, path string) ([]string, map[pullKey]string) {
	var nodeIds []string
	locations := map[pullKey]string{}
	for _, f := range m.Files {
		if !f.FileName.Valid {
			continue
		}
		// Check if the file matches or the folder matches.
		// In both cases, add the package
		curFile := filepath.Join(datasetRoot, f.Path, f.FileName.String)
		curFolder := filepath.Join(datasetRoot, f.Path)
		if curFile != path && curFolder != path {
			continue
		}
		if !slices.Contains(nodeIds, f.PackageNodeId) {
			nodeIds = append(nodeIds, f.PackageNodeId)
		}
		locations[pullKey{f.PackageNodeId, f.FileName.String}] = curFile
	}
	return nodeIds, locations
}

// recordPull notes in mapState that the file at location was pulled.
func recordPull(mapState *models.MapState, datasetRoot, location string, crc32 uint32) {
	relLocation := filepath.ToSlash(strings.TrimPrefix(location, datasetRoot+string(os.PathSeparator)))
	for i, mf := range mapState.Files {
		if mf.Path == relLocation || mf.Path == location {
			mapState.Files[i].PullTime = time.Now()
			mapState.Files[i].Crc32 = crc32
			return
		}
	}

	// First time we pull the file --> create new record in mapState.
	mapState.Files = append(mapState.Files, models.MapStateRecord{
		Path:     relLocation,
		PullTime: time.Now(),
		IsLocal:  true,
		Crc32:    crc32,
	})
}

// findMappedDatasetRoot checks if the provided path is part of a Pennsieve Mapped Dataset.
func findMappedDatasetRoot(startPath string) (string, bool, error) {

	// Remove extension in case the startPath is a file.
	startPath = filepath.FromSlash(startPath)
	parentPath := strings.TrimSuffix(startPath, filepath.Ext(startPath))
	manifestPath := ""
	found := false
	var err error

	for parentPath != "/" && parentPath != "." {

		checkLocation := filepath.Join(parentPath, ".pennsieve", "manifest.json")
		found, err = exists(checkLocation)
		if err != nil {
			return "", found, err
		}
		if found {
			manifestPath = checkLocation
			log.Info(fmt.Sprintf("Found manifest in: %s  ", parentPath))
			break
		}
		nextParent := filepath.Dir(parentPath)
		if nextParent == parentPath {
			// We've reached a filesystem root (e.g., "C:\" on Windows)
			break
		}
		parentPath = nextParent
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
