package api

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/manifest"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/manifest/manifestFile"
	"log"
)

// SyncResponse returns summary info from ManifestSync method.
type SyncResponse struct {
	ManifestNodeId string
	NrFilesUpdated int
	NrFilesRemoved int
	FailedFiles    []string
}

// ManifestSync syncs local manifest with cloud manifest.
func ManifestSync(m *models.Manifest) (*SyncResponse, error) {

	log.Println("MANIFEST SYNC")

	manifestNodeId := ""
	if m.NodeId.Valid {
		manifestNodeId = m.NodeId.String
	}

	client := PennsieveClient

	var f models.ManifestFile

	// Sync all files except those who have status 'Verified'
	requestStatus := []manifestFile.Status{
		manifestFile.Initiated,
		manifestFile.Synced,
		manifestFile.Failed,
		manifestFile.Removed,
		manifestFile.Imported,
		manifestFile.Finalized,
		manifestFile.Unknown,
	}

	offset := 0
	const pageSize = 500
	allResponse := SyncResponse{
		ManifestNodeId: manifestNodeId,
		NrFilesUpdated: int(0),
		NrFilesRemoved: 0,
		FailedFiles:    []string{},
	}

	var allStatusUpdates []manifestFile.FileStatusDTO

	for {
		files, err := f.GetByStatus(m.Id, requestStatus, pageSize, offset)

		if err != nil {
			log.Println("Error getting files for manifest")
		}

		if len(files) == 0 {
			log.Println("Sync complete with offset: ", offset)
			break
		}

		var requestFiles []manifestFile.FileDTO
		for _, file := range files {
			s3Key := fmt.Sprintf("%s/%d", manifestNodeId, f.UploadId)

			reqFile := manifestFile.FileDTO{
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

		log.Println("RESPONSE FROM MANIFEST")
		log.Println(response.UpdatedFiles)

		allResponse.NrFilesUpdated += response.NrFilesUpdated
		allResponse.NrFilesRemoved += response.NrFilesRemoved
		allResponse.FailedFiles = append(allResponse.FailedFiles, response.FailedFiles...)

		allStatusUpdates = append(allStatusUpdates, response.UpdatedFiles...)

		// Update manifestId in table if currently does not exist.
		if !m.NodeId.Valid {
			m.SetManifestNodeId(response.ManifestNodeId)
			m.NodeId = sql.NullString{
				String: response.ManifestNodeId,
				Valid:  true,
			}
			manifestNodeId = response.ManifestNodeId
			allResponse.ManifestNodeId = response.ManifestNodeId
		}

		offset += pageSize
	}

	// Update file status for synchronized manifest.
	f.SyncResponseStatusUpdate(m.Id, allStatusUpdates)

	return &allResponse, nil
}
