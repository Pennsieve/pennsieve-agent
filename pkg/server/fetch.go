package server

import (
    "context"
    "github.com/google/uuid"
    api "github.com/pennsieve/pennsieve-agent/api/v1"
    "github.com/pennsieve/pennsieve-agent/pkg/shared"
    models "github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
    log "github.com/sirupsen/logrus"
    "path/filepath"
)

func (s *server) Fetch(ctx context.Context, req *api.FetchRequest) (*api.SimpleStatusResponse, error) {

    var err error

    // Check if the provided path is part of a mapped dataset
    datasetRoot, found, err := findMappedDatasetRoot(req.Path)
    if err != nil {
        return nil, err
    }

    if !found {
        return &api.SimpleStatusResponse{Status: "The provided path is not part of a Pennsieve mapped dataset."}, nil
    }

    // Read current local Manifest file
    workspaceManifest, err := shared.ReadWorkspaceManifest(filepath.Join(datasetRoot, ".pennsieve", "manifest.json"))
    if err != nil {
        return nil, err
    }

    // Download new Manifest File.
    client := s.client
    manifestResponse, err := client.Dataset.GetManifest(ctx, workspaceManifest.DatasetNodeId)
    if err != nil {
        log.Errorf("Download failed: %v", err)
        return nil, err
    }

    // Download New Manifest .pennsieve folder ---> rename to Manifest-New until finished
    manifestLocation := filepath.Join(datasetRoot, ".pennsieve", "manifest-new.json")
    _, err = s.downloadFileFromPresignedUrl(ctx, manifestResponse.URL, manifestLocation, uuid.New().String())
    if err != nil {
        log.Errorf("Download failed: %v", err)
        return nil, err
    }

    // Now we compare the current and the new manifest. We expect the following changes
    // ADDED: new files were added to new manifest that are not in current manifest
    // DELETED: files are absent from new manifest that were present in current manifest
    // RENAMED: files with same packageId are in new manifest with a new name
    // MOVED: files with the same packageId are in a different path
    //
    // We DO NOT expect any CHANGED files as changes to files would automatically cause a
    // new package file-id/package-id so this would show up as a combination of deleted and
    // added file.
    //
    // We might want to support 'replacing files' in the future and one way of supporting this
    // is to have the same packageID point to a different file with a fileID. Not sure if this
    // will happen but this is why I used the fileID instead of the packageID in some cases.

    resp := &api.SimpleStatusResponse{Status: "Success"}

    return resp, nil
}

func compareCurrentRemoteManifest(datasetRoot, curManifest []models.ManifestDTO, newManifest []models.ManifestDTO) error {

}
