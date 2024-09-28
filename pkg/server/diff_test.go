package server

import (
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateFolderManifest(t *testing.T) {

	files, err := createFolderManifest("../../resources/test")
	assert.NoError(t, err)
	assert.Len(t, files, 4, "Expecting 4 files)")

	// find file4 and check path
	for _, f := range files {
		if f.FileName == "file_4.txt" {
			assert.Equal(t, f.Path, "folder_1/folder_2")
		}
	}

}

func TestCompareManifest(t *testing.T) {
	datasetRoot := "../../resources/test"
	files, _ := createFolderManifest(datasetRoot)

	manifest, err := shared.ReadWorkspaceManifest("../../resources/test/.pennsieve/manifest_old.json")
	assert.NoError(t, err)

	var result = &api.MapStatusResponse{}
	compareManifestToFolder(datasetRoot, manifest.Files, files, result)

	assert.Len(t, result.Files, 4, "Expecting 4 files")
	sanityCount := 0
	for _, f := range result.Files {
		if f.Content.Name == "file_6.txt" {
			assert.Equal(t, f.ChangeType, api.PackageStatus_DELETED)
			sanityCount++
		}

		if f.Content.Name == "file_1.txt" {
			assert.Equal(t, f.ChangeType, api.PackageStatus_ADDED)
			sanityCount++
		}
	}

	assert.Equal(t, 2, sanityCount, "Expect that both if statements are triggered once.")
}


