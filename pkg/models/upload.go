package models

import "github.com/pennsieve/pennsieve-go-core/pkg/models/manifest/manifestFile"

type UploadStatusUpdateMessage struct {
	UploadID string
	Status   manifestFile.Status
}
