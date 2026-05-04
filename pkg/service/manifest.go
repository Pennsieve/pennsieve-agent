package service

import (
	"context"
	"database/sql"

	"github.com/pennsieve/pennsieve-agent/pkg/models"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/manifest/manifestFile"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
)

type ManifestService struct {
	mStore  store.ManifestStore
	mfStore store.ManifestFileStore
	client  *pennsieve.Client
}

func NewManifestService(ms store.ManifestStore, mfs store.ManifestFileStore, client *pennsieve.Client) *ManifestService {
	return &ManifestService{
		mStore:  ms,
		mfStore: mfs,
		client:  client,
	}
}

func (s *ManifestService) GetManifest(manifestId int32) (*store.Manifest, error) {
	return s.mStore.Get(manifestId)
}

func (s *ManifestService) GetAll() ([]store.Manifest, error) {
	return s.mStore.GetAll()
}

func (s *ManifestService) Add(params store.ManifestParams) (*store.Manifest, error) {
	return s.mStore.Add(params)
}

func (s *ManifestService) RemoveFromManifest(manifestId int32, removePath string) (models.RemoveFromManifestResponse, error) {
	return s.mfStore.RemoveFromManifest(manifestId, removePath)
}

func (s *ManifestService) RemoveManifest(manifestId int32) error {
	return s.mStore.Remove(manifestId)
}

func (s *ManifestService) GetFiles(manifestId int32, limit int32, offset int32) ([]store.ManifestFile, error) {
	return s.mfStore.Get(manifestId, limit, offset)
}

func (s *ManifestService) ResetStatusForManifest(manifestId int32) error {
	return s.mfStore.ResetStatusForManifest(manifestId)
}

func (s *ManifestService) GetNumberOfRowsForStatus(manifestId int32, statusArr []manifestFile.Status, invert bool) (int64, error) {
	return s.mfStore.GetNumberOfRowsForStatus(manifestId, statusArr, invert)
}

func (s *ManifestService) ManifestFilesToChannel(ctx context.Context, manifestId int32, statusArr []manifestFile.Status, walker chan<- store.ManifestFile) {
	s.mfStore.ManifestFilesToChannel(ctx, manifestId, statusArr, walker)
}

func (s *ManifestService) SyncResponseStatusUpdate(manifestId int32, statusList []manifestFile.FileStatusDTO) error {
	return s.mfStore.SyncResponseStatusUpdate(manifestId, statusList)
}

// SetManifestNodeId updates the manifest Node ID in the Manifest object and Database
func (s *ManifestService) SetManifestNodeId(m *store.Manifest, nodeId string) error {

	m.NodeId = sql.NullString{
		String: nodeId,
		Valid:  true,
	}

	return s.mStore.SetManifestNodeId(m.Id, nodeId)
}

func (s *ManifestService) AddFiles(records []store.ManifestFileParams) error {
	return s.mfStore.Add(records)
}

func (s *ManifestService) SetFileStatus(uploadId string, status manifestFile.Status) error {
	return s.mfStore.SetStatus(status, uploadId)
}

func (s *ManifestService) BatchSetFileStatus(uploadIds []string, status manifestFile.Status) error {
	return s.mfStore.BatchSetStatus(status, uploadIds)
}
