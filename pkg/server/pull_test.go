package server

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
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
