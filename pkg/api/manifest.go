package api

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-go-api/models/manifest"
	"log"
)

// SyncResponse returns summary info from ManifestSync method.
type SyncResponse struct {
	ManifestNodeId string
	NrFilesUpdated int32
	NrFilesRemoved int32
	FailedFiles    []string
}

// ManifestSync syncs local manifest with cloud manifest.
func ManifestSync(m *models.Manifest) (*SyncResponse, error) {

	manifestNodeId := ""
	if m.NodeId.Valid {
		manifestNodeId = m.NodeId.String
	}

	client := PennsieveClient

	var f models.ManifestFile
	requestStatus := []manifest.ManifestFileStatus{
		manifest.FileInitiated,
		manifest.FileFailed,
		manifest.FileRemoved,
	}

	//TODO: paginate
	files, err := f.GetByStatus(m.Id, requestStatus, 1000, 0)

	var requestFiles []manifest.FileDTO
	for _, file := range files {
		s3Key := fmt.Sprintf("%s/%d", manifestNodeId, f.UploadId)

		reqFile := manifest.FileDTO{
			UploadID:   file.UploadId.String(),
			S3Key:      s3Key,
			TargetPath: file.TargetPath,
			TargetName: file.TargetName,
			Status:     file.Status,
		}
		requestFiles = append(requestFiles, reqFile)
	}

	fmt.Println("Number of Files: ", len(requestFiles))

	requestBody := manifest.DTO{
		DatasetId: m.DatasetId,
		ID:        manifestNodeId,
		Files:     requestFiles,
		Status:    m.Status,
	}

	response, err := client.Manifest.Create(context.Background(), requestBody)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// Update manifestId in table if currently does not exist.
	if !m.NodeId.Valid {
		m.SetManifestNodeId(response.ManifestNodeId)
		m.NodeId = sql.NullString{
			String: response.ManifestNodeId,
			Valid:  true,
		}
	}

	// Update file status for synchronized manifest.
	f.SyncResponseStatusUpdate(m.Id, response.FailedFiles)

	resp := SyncResponse{
		ManifestNodeId: response.ManifestNodeId,
		NrFilesUpdated: int32(response.NrFilesUpdated),
		NrFilesRemoved: int32(response.NrFilesRemoved),
		FailedFiles:    response.FailedFiles,
	}

	return &resp, nil
}
