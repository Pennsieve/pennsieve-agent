package server

import (
	"database/sql"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	models "github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
	"github.com/stretchr/testify/assert"
	"path"
	"path/filepath"
	"testing"
)

func TestCompareManifest(t *testing.T) {

	absPath, _ := filepath.Abs(".")
	root := filepath.Dir(filepath.Dir(absPath))
	datasetRoot := filepath.ToSlash(filepath.Join(root, "resources", "test", "mapDataset"))

	files, _ := createFolderManifest(datasetRoot)

	manifest, err := shared.ReadWorkspaceManifest(filepath.Join(datasetRoot, ".pennsieve", "manifest.json"))
	assert.NoError(t, err)

	result, err := compareManifestToFolder(datasetRoot, manifest.Files, files)
	assert.NoError(t, err)

	assert.Len(t, result, 9, "Expect 9 results to have changes compared to original manifest.")

	count := 0
	for _, f := range result {
		switch f.FilePath {
		case path.Join("folder_1", "file_10_added.txt"):
			assert.Equal(t, api.PackageStatus_ADDED, f.Type)
			assert.Equal(t, f.New.FileName, "file_10_added.txt")
			count++
		case path.Join("folder_1", "file_4_renamed.txt"):
			assert.Equal(t, api.PackageStatus_RENAMED, f.Type)
			assert.Equal(t, f.New.FileName, "file_4_renamed.txt")
			assert.Equal(t, f.Old.FileName, "file_4.txt")
			count++
		case path.Join("folder_1", "folder_1_1", "file_5_moved.txt"):
			assert.Equal(t, api.PackageStatus_MOVED, f.Type)
			assert.Equal(t, f.New.Path, path.Join("folder_1", "folder_1_1"))
			assert.Equal(t, f.Old.Path, "folder_1")
			count++
		case path.Join("folder_2", "file_7_downloaded_renamed.txt"):
			assert.Equal(t, api.PackageStatus_RENAMED, f.Type)
			assert.Equal(t, f.New.FileName, "file_7_downloaded_renamed.txt")
			assert.Equal(t, f.Old.FileName, "file_7_downloaded.txt")
			count++
		case path.Join("folder_2", "folder_2_1", "file_12_downloaded_moved_renamed.txt"):
			assert.Equal(t, api.PackageStatus_MOVED_RENAMED, f.Type)
			assert.Equal(t, f.New.FileName, "file_12_downloaded_moved_renamed.txt")
			assert.Equal(t, f.Old.FileName, "file_12_downloaded.txt")
			assert.Equal(t, f.New.Path, path.Join("folder_2", "folder_2_1"))
			assert.Equal(t, f.Old.Path, "folder_2")
			count++
		case path.Join("folder_2", "folder_2_1", "file_13_moved_renamed.txt"):
			assert.Equal(t, api.PackageStatus_MOVED_RENAMED, f.Type)
			assert.Equal(t, f.New.FileName, "file_13_moved_renamed.txt")
			assert.Equal(t, f.Old.FileName, "file_13.txt")
			assert.Equal(t, f.New.Path, path.Join("folder_2", "folder_2_1"))
			assert.Equal(t, f.Old.Path, "folder_2")
			count++
		case path.Join("folder_2", "folder_2_1", "file_8_downloaded_moved.txt"):
			assert.Equal(t, api.PackageStatus_MOVED, f.Type)
			assert.Equal(t, f.New.Path, path.Join("folder_2", "folder_2_1"))
			assert.Equal(t, f.Old.Path, "folder_2")
			count++
		case path.Join("folder_2", "file_9_downloaded_changed.txt"):
			assert.Equal(t, api.PackageStatus_CHANGED, f.Type)
			assert.Equal(t, f.Changed.from.Size, models.NullInt{NullInt64: sql.NullInt64{Valid: true, Int64: 35}})
			assert.Equal(t, f.Changed.Size, int64(44))
			count++
		case path.Join("folder_2", "file_14_deleted.txt"):
			assert.Equal(t, api.PackageStatus_DELETED, f.Type)
			count++

		}

	}

	assert.Equal(t, 9, count, "Expect each case statement in switch to be called once.")

}
