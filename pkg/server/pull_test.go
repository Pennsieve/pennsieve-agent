package server

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pennsieve/pennsieve-agent/v2/pkg/models"
	wsmodels "github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
	"github.com/stretchr/testify/assert"
)

func TestFindMappedDatasetRoot(t *testing.T) {

	root, found, err := findMappedDatasetRoot(filepath.Join("..", "..", "resources", "test", "testData", "pullTest", "folder_1", "folder_2"))
	assert.NoError(t, err)
	assert.True(t, found)
	_, lastPathName := filepath.Split(root)
	assert.Equal(t, "pullTest", lastPathName,
		"Should find the manifest file in root folder when starting in subfolder in mapped dataset.")

	root, found, err = findMappedDatasetRoot(filepath.Join("..", "..", "resources", "test", "testData", "pullTest"))
	assert.NoError(t, err)
	assert.True(t, found,
		"Should find the manifest file in root folder when starting at root")

	root, found, err = findMappedDatasetRoot(filepath.Join("..", "..", "resources"))
	assert.NoError(t, err)
	assert.False(t, found,
		"Should not find a root if not starting in mapped dataset")

	root, found, err = findMappedDatasetRoot(filepath.Join("..", "..", "resources", "test", "testData", "pullTest", "folder_1", "folder_2", "file_4.txt"))
	assert.NoError(t, err)
	assert.True(t, found,
		"Should find manifest when input is a file-name")
}

func TestPullSelection(t *testing.T) {
	root := filepath.FromSlash("/data/ds")
	m := &wsmodels.WorkspaceManifest{Files: []wsmodels.ManifestDTO{
		{PackageNodeId: "N:package:1", FileName: wsmodels.NullString{NullString: sql.NullString{String: "a.csv", Valid: true}}, Path: "study"},
		{PackageNodeId: "N:package:2", FileName: wsmodels.NullString{NullString: sql.NullString{String: "b1.edf", Valid: true}}, Path: "study"},
		{PackageNodeId: "N:package:2", FileName: wsmodels.NullString{NullString: sql.NullString{String: "b2.edf", Valid: true}}, Path: "study"},
		{PackageNodeId: "N:package:3", FileName: wsmodels.NullString{NullString: sql.NullString{String: "c.csv", Valid: true}}, Path: "other"},
		{PackageNodeId: "N:package:4", Path: "study"},
	}}

	nodeIds, locations := pullSelection(m, root, filepath.Join(root, "study"))
	assert.Equal(t, []string{"N:package:1", "N:package:2"}, nodeIds, "a folder pulls its packages, each once")
	assert.Equal(t, filepath.Join(root, "study", "b2.edf"), locations[pullKey{"N:package:2", "b2.edf"}],
		"each file of a package has its own place")
	assert.Len(t, locations, 3)

	nodeIds, _ = pullSelection(m, root, filepath.Join(root, "other", "c.csv"))
	assert.Equal(t, []string{"N:package:3"}, nodeIds, "a file pulls its package")
}

func TestRecordPull(t *testing.T) {
	root := filepath.FromSlash("/data/ds")
	state := &models.MapState{Files: []models.MapStateRecord{{Path: "study/a.csv", Crc32: 1}}}

	recordPull(state, root, filepath.Join(root, "study", "a.csv"), 7)
	recordPull(state, root, filepath.Join(root, "study", "b.csv"), 8)

	assert.Len(t, state.Files, 2)
	assert.Equal(t, uint32(7), state.Files[0].Crc32, "a file pulled before is updated")
	assert.Equal(t, models.MapStateRecord{Path: "study/b.csv", PullTime: state.Files[1].PullTime, IsLocal: true, Crc32: 8}, state.Files[1])
}
