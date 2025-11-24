package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	log "github.com/sirupsen/logrus"
)

// Push identifies new files in a mapped dataset and uploads them to Pennsieve
func (s *agentServer) Push(ctx context.Context, req *api.PushRequest) (*api.SimpleStatusResponse, error) {

	// Check if the provided path is part of a mapped dataset
	datasetRoot, found, err := findMappedDatasetRoot(req.Path)
	if err != nil {
		return nil, err
	}

	if !found {
		return &api.SimpleStatusResponse{Status: "The provided path is not part of a Pennsieve mapped dataset."}, nil
	}

	manifestPath := filepath.Join(datasetRoot, ".pennsieve", "manifest.json")

	// Use existing diff functionality to find local additions
	diffResp, err := s.GetMapDiff(ctx, &api.MapDiffRequest{Path: datasetRoot})
	if err != nil {
		log.Errorf("Unable to calculate diff for push: %v", err)
		return nil, err
	}

	var newFiles []string
	seen := make(map[string]struct{})
	for _, fileStatus := range diffResp.GetFiles() {
		if fileStatus.GetChangeType() != api.PackageStatus_ADDED {
			continue
		}

		content := fileStatus.GetContent()
		if content == nil {
			continue
		}

		relDir := filepath.FromSlash(content.GetPath())
		absPath := filepath.Join(datasetRoot, relDir, content.GetName())

		if _, ok := seen[absPath]; ok {
			continue
		}
		seen[absPath] = struct{}{}
		newFiles = append(newFiles, absPath)
	}

	if len(newFiles) == 0 {
		return &api.SimpleStatusResponse{Status: "No new files to push."}, nil
	}

	log.Infof("Found %d new file(s) to push", len(newFiles))

	// Create manifest and add files
	// This runs in a goroutine to prevent blocking
	go func() {
		// Create an empty manifest first
		manifestResponse, err := s.ManifestService().Add(getManifestParams(s))
		if err != nil {
			log.Errorf("Error creating manifest: %v", err)
			s.messageSubscribers(fmt.Sprintf("Error creating manifest: %v", err))
			return
		}

		log.Infof("Created manifest %d for push", manifestResponse.Id)
		s.messageSubscribers(fmt.Sprintf("Created manifest %d with %d file(s) to push", manifestResponse.Id, len(newFiles)))

		// Add each file to the manifest individually with its correct target path
		successCount := 0
		for _, filePath := range newFiles {
			// Get the relative path from dataset root
			relPath, err := filepath.Rel(datasetRoot, filePath)
			if err != nil {
				log.Errorf("Error computing relative path for %s: %v", filePath, err)
				continue
			}

			// The target path is the directory part of the relative path
			targetPath := filepath.Dir(relPath)
			if targetPath == "." {
				targetPath = ""
			}
			// Convert to forward slashes for target path
			targetPath = filepath.ToSlash(targetPath)

			// Add this file to the manifest using the existing addToManifest helper
			_, err = s.addToManifest(filePath, targetPath, nil, manifestResponse.Id)
			if err != nil {
				log.Errorf("Error adding file %s to manifest: %v", relPath, err)
				continue
			}
			successCount++
		}

		log.Infof("Added %d file(s) to manifest %d", successCount, manifestResponse.Id)
		s.messageSubscribers(fmt.Sprintf("Added %d file(s) to manifest %d", successCount, manifestResponse.Id))

		// Upload the manifest
		uploadReq := api.UploadManifestRequest{
			ManifestId: manifestResponse.Id,
		}

		_, err = s.UploadManifest(context.Background(), &uploadReq)
		if err != nil {
			log.Errorf("Error uploading manifest %d: %v", manifestResponse.Id, err)
			s.messageSubscribers(fmt.Sprintf("Error uploading manifest: %v", err))
			return
		}

		s.messageSubscribers(fmt.Sprintf("Upload initiated for manifest %d", manifestResponse.Id))
		log.Infof("Push complete for manifest %d", manifestResponse.Id)

		// Update local workspace manifest to include newly uploaded files
		if err := updateLocalManifest(manifestPath, datasetRoot, newFiles); err != nil {
			log.Errorf("Error updating local manifest: %v", err)
			// Don't return - the upload was successful, just log the error
		} else {
			log.Infof("Updated local manifest with %d new file(s)", len(newFiles))
		}
	}()

	resp := &api.SimpleStatusResponse{Status: fmt.Sprintf("Push initiated for %d file(s). Use \"pennsieve agent subscribe\" to track progress.", len(newFiles))}
	return resp, nil
}

// getManifestParams is a helper to get manifest parameters for creating a new manifest
func getManifestParams(s *agentServer) store.ManifestParams {
	// This is a simplified version - you may need to get actual user/dataset info
	// Similar to how it's done in CreateManifest
	activeUser, _ := s.UserService().GetActiveUser()
	curClientSession, _ := s.UserService().GetUserSettings()

	// Get dataset info
	client, _ := s.PennsieveClient()
	ds, _ := client.Dataset.Get(context.Background(), curClientSession.UseDatasetId)

	return store.ManifestParams{
		UserId:           activeUser.Id,
		UserName:         activeUser.Name,
		OrganizationId:   activeUser.OrganizationId,
		OrganizationName: activeUser.OrganizationName,
		DatasetId:        curClientSession.UseDatasetId,
		DatasetName:      ds.Content.Name,
	}
}

// updateLocalManifest adds newly uploaded files to the local workspace manifest
// Note: Updating local manifest doesn't create a package ID or a checksum
func updateLocalManifest(manifestPath string, datasetRoot string, newFiles []string) error {
	// Get the current manifest
	workspaceManifest, err := shared.ReadWorkspaceManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read workspace manifest: %w", err)
	}

	// Add new files
	for _, filePath := range newFiles {
		// Get file info
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			log.Warnf("Cannot stat file %s: %v", filePath, err)
			continue
		}

		// Get the relative path from dataset root
		relPath, err := filepath.Rel(datasetRoot, filePath)
		if err != nil {
			log.Warnf("Cannot compute relative path for %s: %v", filePath, err)
			continue
		}

		// Split into directory path and filename
		dirPath := filepath.Dir(relPath)
		if dirPath == "." {
			dirPath = ""
		}
		// Convert to forward slashes for consistency
		dirPath = filepath.ToSlash(dirPath)
		fileName := filepath.Base(relPath)

		manifestEntry := shared.CreateManifestDTO(fileName, dirPath, fileInfo.Size())

		// Append to the files list
		workspaceManifest.Files = append(workspaceManifest.Files, manifestEntry)
	}

	// Write the updated manifest back to disk
	if err := shared.WriteWorkspaceManifest(manifestPath, workspaceManifest); err != nil {
		return fmt.Errorf("failed to write workspace manifest: %w", err)
	}

	return nil
}
