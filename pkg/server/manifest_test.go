package server

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRecordCreation(t *testing.T) {

	paths := []string{
		"/Users/user1/testUpload/.DS_Store",
		"/Users/user1/testUpload/Folder 1/.DS_Store",
		"/Users/user1/testUpload/File 1.png",
		"/Users/user1/testUpload/Folder 1/Folder 1 - File 1.png",
		"/Users/user1/testUpload/file.with.many.periods.png",
		"/Users/user1/testUpload/file ,.\\[]@#$%@~.",
	}

	localBasePath := "/Users/user1/testUpload"
	targetBasePath := ""
	manifestId := int32(1)

	records := recordsFromPaths(paths, localBasePath, targetBasePath, manifestId)

	assert.Equal(t, int32(1), records[0].ManifestId, "ManifestID must match.")

	assert.Equal(t, ".DS_Store", records[0].TargetName,
		"Hidden file name in root folder does not match expected value.")
	assert.Equal(t, ".DS_Store", records[1].TargetName,
		"Hidden file name in sub folder does not match expected value.")

	assert.Equal(t, "File 1.png", records[2].TargetName,
		"File name in root folder does not match expected value.")
	assert.Equal(t, "Folder 1 - File 1.png", records[3].TargetName,
		"File name in sub folder does not match expected value.")

	assert.Equal(t, "", records[2].TargetPath,
		"File in root folder should have empty target path.")
	assert.Equal(t, "Folder 1", records[3].TargetPath,
		"File in sub folder should have sub folder as path.")

	assert.Equal(t, "file.with.many.periods.png", records[4].TargetName,
		"Record should handle files with multiple periods.")
	assert.Equal(t, "file ,.\\[]@#$%@~.", records[5].TargetName,
		"Record should handle ood characters")

	// Rerun with new target base path.
	targetBasePath = "newTargetPath"
	records = recordsFromPaths(paths, localBasePath, targetBasePath, manifestId)

	assert.Equal(t, "File 1.png", records[2].TargetName,
		"File name in targetPath folder does not match expected value.")
	assert.Equal(t, "newTargetPath", records[2].TargetPath,
		"TargetPath should match targetBasePath")
	assert.Equal(t, "newTargetPath/Folder 1", records[3].TargetPath,
		"TargetBasePath should be root of target path.")
}
