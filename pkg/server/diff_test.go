package server

import (
	"database/sql"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	models "github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestCompareManifest(t *testing.T) {
	datasetRoot, err := filepath.Abs(filepath.Join("..", "..", "resources", "test", "mapDataset"))
	assert.NoError(t, err)

	files, _ := createFolderManifest(datasetRoot)

	manifestLoc, _ := filepath.Abs(filepath.Join("resources", "test", "mapDataset", ".pennsieve", "manifest.json"))

	manifest, err := shared.ReadWorkspaceManifest(manifestLoc)
	assert.NoError(t, err)

	result, err := compareManifestToFolder(datasetRoot, manifest.Files, files)
	assert.NoError(t, err)

	assert.Len(t, result, 8, "Expect 8 results to have changes compared to original manifest.")

	count := 0
	for _, f := range result {
		switch f.FilePath {
		case filepath.Join("folder_1", "file_10_added.txt"):
			assert.Equal(t, api.PackageStatus_ADDED, f.Type)
			assert.Equal(t, f.New.FileName, "file_10_added.txt")
			count++
		case filepath.Join("folder_1", "file_4_renamed.txt"):
			assert.Equal(t, api.PackageStatus_RENAMED, f.Type)
			assert.Equal(t, f.New.FileName, "file_4_renamed.txt")
			assert.Equal(t, f.Old.FileName, "file_4.txt")
			count++
		case filepath.Join("folder_1", "folder_1_1", "file_5_moved.txt"):
			assert.Equal(t, api.PackageStatus_MOVED, f.Type)
			assert.Equal(t, f.New.Path, "folder_1/folder_1_1")
			assert.Equal(t, f.Old.Path, "folder_1")
			count++
		case filepath.Join("folder_2", "file_7_downloaded_renamed.txt"):
			assert.Equal(t, api.PackageStatus_RENAMED, f.Type)
			assert.Equal(t, f.New.FileName, "file_7_downloaded_renamed.txt")
			assert.Equal(t, f.Old.FileName, "file_7_downloaded.txt")
			count++
		case filepath.Join("folder_2", "folder_2_1", "file_12_downloaded_moved_renamed.txt"):
			assert.Equal(t, api.PackageStatus_MOVED_RENAMED, f.Type)
			assert.Equal(t, f.New.FileName, "file_12_downloaded_moved_renamed.txt")
			assert.Equal(t, f.Old.FileName, "file_12_downloaded.txt")
			assert.Equal(t, f.New.Path, "folder_2/folder_2_1")
			assert.Equal(t, f.Old.Path, "folder_2")
			count++
		case filepath.Join("folder_2", "folder_2_1", "file_13_moved_renamed.txt"):
			assert.Equal(t, api.PackageStatus_MOVED_RENAMED, f.Type)
			assert.Equal(t, f.New.FileName, "file_13_moved_renamed.txt")
			assert.Equal(t, f.Old.FileName, "file_13.txt")
			assert.Equal(t, f.New.Path, "folder_2/folder_2_1")
			assert.Equal(t, f.Old.Path, "folder_2")
			count++
		case filepath.Join("folder_2", "folder_2_1", "file_8_downloaded_moved.txt"):
			assert.Equal(t, api.PackageStatus_MOVED, f.Type)
			assert.Equal(t, f.New.Path, "folder_2/folder_2_1")
			assert.Equal(t, f.Old.Path, "folder_2")
			count++
		case filepath.Join("folder_2", "file_9_downloaded_changed.txt"):
			assert.Equal(t, api.PackageStatus_CHANGED, f.Type)
			assert.Equal(t, f.Changed.from.Size, models.NullInt{NullInt64: sql.NullInt64{Valid: true, Int64: 35}})
			assert.Equal(t, f.Changed.Size, int64(44))
			count++
		}
	}

	assert.Equal(t, 8, count, "Expect each case statement in switch to be called once.")

}
