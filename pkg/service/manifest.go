package service

import (
	"context"
	"database/sql"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/manifest/manifestFile"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	log "github.com/sirupsen/logrus"
)

type ManifestService struct {
	mStore  store.ManifestStore
	mfStore store.ManifestFileStore
	client  *pennsieve.Client
}

func NewManifestService(ms store.ManifestStore, mfs store.ManifestFileStore) *ManifestService {
	return &ManifestService{
		mStore:  ms,
		mfStore: mfs,
	}
}

func (s *ManifestService) SetPennsieveClient(client *pennsieve.Client) {
	s.client = client
}

// VerifyFinalizedStatus checks if files are in "Finalized" state on server and sets to "Verified"
func (s *ManifestService) VerifyFinalizedStatus(manifest *store.Manifest) error {
	log.Debug("Verifying files")

	response, err := s.client.Manifest.GetFilesForStatus(nil, manifest.NodeId.String, manifestFile.Finalized, "", true)
	if err != nil {
		log.Error("Error getting files for status, here is why: ", err)
		return err
	}

	log.Debug("Number of responses: ", len(response.Files))
	if len(response.Files) > 0 {
		if len(response.Files) == 1 {
			s.mfStore.SetStatus(manifestFile.Verified, response.Files[0])
		} else {
			s.mfStore.BatchSetStatus(manifestFile.Verified, response.Files)
		}
	}

	for {
		if len(response.ContinuationToken) > 0 {
			log.Debug("Getting another set of files ")
			response, err = s.client.Manifest.GetFilesForStatus(nil, manifest.NodeId.String, manifestFile.Finalized, response.ContinuationToken, true)
			if err != nil {
				log.Error("Error getting files for status, here is why: ", err)
				return err
			}
			if len(response.Files) > 0 {
				if len(response.Files) == 1 {
					s.mfStore.SetStatus(manifestFile.Verified, response.Files[0])
				} else {
					s.mfStore.BatchSetStatus(manifestFile.Verified, response.Files)
				}
			}
		} else {
			break
		}
	}

	return nil
}

func (s *ManifestService) GetManifest(manifestId int32) (*store.Manifest, error) {
	manifest, err := s.mStore.Get(manifestId)
	return manifest, err
}

func (s *ManifestService) GetAll() ([]store.Manifest, error) {
	manifests, err := s.mStore.GetAll()
	return manifests, err
}

func (s *ManifestService) Add(params store.ManifestParams) (*store.Manifest, error) {
	manifest, err := s.mStore.Add(params)
	return manifest, err
}

func (s *ManifestService) RemoveFromManifest(manifestId int32, removePath string) error {
	err := s.mfStore.RemoveFromManifest(manifestId, removePath)
	return err
}

func (s *ManifestService) RemoveManifest(manifestId int32) error {
	err := s.mStore.Remove(manifestId)
	return err
}

func (s *ManifestService) GetFiles(manifestId int32, limit int32, offset int32) ([]store.ManifestFile, error) {
	files, err := s.mfStore.Get(manifestId, limit, offset)
	return files, err
}

func (s *ManifestService) ResetStatusForManifest(manifestId int32) error {
	err := s.mfStore.ResetStatusForManifest(manifestId)
	return err
}

func (s *ManifestService) GetNumberOfRowsForStatus(manifestId int32, statusArr []manifestFile.Status, invert bool) (int64, error) {
	result, err := s.mfStore.GetNumberOfRowsForStatus(manifestId, statusArr, invert)
	return result, err

}

func (s *ManifestService) ManifestFilesToChannel(ctx context.Context, manifestId int32, statusArr []manifestFile.Status, walker chan<- store.ManifestFile) {
	s.mfStore.ManifestFilesToChannel(ctx, manifestId, statusArr, walker)
}

func (s *ManifestService) SyncResponseStatusUpdate(manifestId int32, statusList []manifestFile.FileStatusDTO) error {
	err := s.mfStore.SyncResponseStatusUpdate(manifestId, statusList)
	return err
}

// SetManifestNodeId updates the manifest Node ID in the Manifest object and Database
func (s *ManifestService) SetManifestNodeId(m *store.Manifest, nodeId string) error {

	m.NodeId = sql.NullString{
		String: nodeId,
		Valid:  true,
	}

	err := s.mStore.SetManifestNodeId(m.Id, nodeId)
	return err
}

func (s *ManifestService) AddFiles(records []store.ManifestFileParams) error {
	err := s.mfStore.Add(records)
	return err
}

func (s *ManifestService) SetFileStatus(uploadId string, status manifestFile.Status) error {
	err := s.mfStore.SetStatus(status, uploadId)
	return err
}
