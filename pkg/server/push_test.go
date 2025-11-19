package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPush_FindMappedDatasetRoot(t *testing.T) {
	// Create a temporary directory structure with manifest
	tempDir := t.TempDir()
	pennsieveDir := filepath.Join(tempDir, ".pennsieve")
	manifestPath := filepath.Join(pennsieveDir, "manifest.json")
	subdir1 := filepath.Join(tempDir, "folder_1")
	subdir2 := filepath.Join(subdir1, "folder_2")
	testFile := filepath.Join(subdir2, "the_great_hunt.txt")

	// Create directory structure
	require.NoError(t, os.MkdirAll(pennsieveDir, 0755))
	require.NoError(t, os.MkdirAll(subdir2, 0755))
	require.NoError(t, os.WriteFile(manifestPath, []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

	// Test finding root from subfolder
	root, found, err := findMappedDatasetRoot(subdir2)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, tempDir, root,
		"Should find the manifest file in root folder when starting in subfolder in mapped dataset.")

	// Test finding root from root folder
	root, found, err = findMappedDatasetRoot(tempDir)
	assert.NoError(t, err)
	assert.True(t, found,
		"Should find the manifest file in root folder when starting at root")
	assert.Equal(t, tempDir, root)

	// Test not finding root when not in mapped dataset
	tempDirNoManifest := t.TempDir()
	root, found, err = findMappedDatasetRoot(tempDirNoManifest)
	assert.NoError(t, err)
	assert.False(t, found,
		"Should not find a root if not starting in mapped dataset")

	// Test finding root when input is a file path
	root, found, err = findMappedDatasetRoot(testFile)
	assert.NoError(t, err)
	assert.True(t, found,
		"Should find manifest when input is a file-name")
	assert.Equal(t, tempDir, root)
}

func TestPush_NotInMappedDataset(t *testing.T) {
	// Create a temporary directory with no manifest
	tempDir := t.TempDir()

	// Create a mock agent server (minimal setup)
	server := &agentServer{}

	// Create push request for directory without manifest
	req := &api.PushRequest{
		Path: tempDir,
	}

	// Execute push
	resp, err := server.Push(context.Background(), req)

	// Should not error, but should return message about not being in mapped dataset
	assert.NoError(t, err)
	assert.Contains(t, resp.Status, "not part of a Pennsieve mapped dataset")
}

func TestPush_NoNewFiles(t *testing.T) {
	// Create a temporary directory structure with manifest
	tempDir := t.TempDir()
	pennsieveDir := filepath.Join(tempDir, ".pennsieve")
	manifestPath := filepath.Join(pennsieveDir, "manifest.json")

	// Create .pennsieve directory
	require.NoError(t, os.MkdirAll(pennsieveDir, 0755))

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

	// Create manifest that includes the test file
	manifest := map[string]interface{}{
		"datasetNodeId":      "N:dataset:test-123",
		"organizationNodeId": "N:organization:test-456",
		"files": []map[string]interface{}{
			{
				"packageId":   "N:package:test-789",
				"packageName": "test.txt",
				"fileId":      "test-789",
				"fileName":    "test.txt",
				"path":        "",
				"size":        12,
			},
		},
	}

	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, manifestJSON, 0644))

	// Create a mock agent server
	server := &agentServer{}

	// Create push request
	req := &api.PushRequest{
		Path: tempDir,
	}

	// Execute push
	resp, err := server.Push(context.Background(), req)

	// Should not error and should report no new files
	assert.NoError(t, err)
	assert.Contains(t, resp.Status, "No new files to push")
}

func TestPush_WithNewFiles(t *testing.T) {
	// Create a temporary directory structure with manifest
	tempDir := t.TempDir()
	pennsieveDir := filepath.Join(tempDir, ".pennsieve")
	manifestPath := filepath.Join(pennsieveDir, "manifest.json")

	// Create .pennsieve directory
	require.NoError(t, os.MkdirAll(pennsieveDir, 0755))

	// Create an existing file (in manifest)
	existingFile := filepath.Join(tempDir, "existing.txt")
	require.NoError(t, os.WriteFile(existingFile, []byte("existing content"), 0644))

	// Create a new file (not in manifest)
	newFile := filepath.Join(tempDir, "new.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("new content"), 0644))

	// Create manifest that only includes existing file
	manifest := map[string]interface{}{
		"datasetNodeId":      "N:dataset:test-123",
		"organizationNodeId": "N:organization:test-456",
		"files": []map[string]interface{}{
			{
				"packageId":   "N:package:test-789",
				"packageName": "existing.txt",
				"fileId":      "test-789",
				"fileName":    "existing.txt",
				"path":        "",
				"size":        16,
			},
		},
	}

	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, manifestJSON, 0644))

	// Create a mock agent server
	server := &agentServer{}

	// Create push request
	req := &api.PushRequest{
		Path: tempDir,
	}

	// Execute push
	resp, err := server.Push(context.Background(), req)

	// Should not error and should report files to push
	assert.NoError(t, err)
	assert.Contains(t, resp.Status, "Push initiated")
	assert.Contains(t, resp.Status, "1 file")
}

func TestPush_IgnoresSystemFiles(t *testing.T) {
	// Create a temporary directory structure with manifest
	tempDir := t.TempDir()
	pennsieveDir := filepath.Join(tempDir, ".pennsieve")
	manifestPath := filepath.Join(pennsieveDir, "manifest.json")

	// Create .pennsieve directory
	require.NoError(t, os.MkdirAll(pennsieveDir, 0755))

	// Create a .DS_Store file (should be ignored)
	dsStoreFile := filepath.Join(tempDir, ".DS_Store")
	require.NoError(t, os.WriteFile(dsStoreFile, []byte("mac system file"), 0644))

	// Create manifest with no files
	manifest := map[string]interface{}{
		"datasetNodeId":      "N:dataset:test-123",
		"organizationNodeId": "N:organization:test-456",
		"files":              []interface{}{},
	}

	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, manifestJSON, 0644))

	// Create a mock agent server
	server := &agentServer{}

	// Create push request
	req := &api.PushRequest{
		Path: tempDir,
	}

	// Execute push
	resp, err := server.Push(context.Background(), req)

	// Should not error and should report no new files (DS_Store ignored)
	assert.NoError(t, err)
	assert.Contains(t, resp.Status, "No new files to push")
}

func TestPush_SkipsPennsieveFolder(t *testing.T) {
	// Create a temporary directory structure with manifest
	tempDir := t.TempDir()
	pennsieveDir := filepath.Join(tempDir, ".pennsieve")
	manifestPath := filepath.Join(pennsieveDir, "manifest.json")

	// Create .pennsieve directory
	require.NoError(t, os.MkdirAll(pennsieveDir, 0755))

	// Create a file inside .pennsieve folder (should be ignored)
	pennsieveFile := filepath.Join(pennsieveDir, "some-file.json")
	require.NoError(t, os.WriteFile(pennsieveFile, []byte("internal file"), 0644))

	// Create manifest with no files
	manifest := map[string]interface{}{
		"datasetNodeId":      "N:dataset:test-123",
		"organizationNodeId": "N:organization:test-456",
		"files":              []interface{}{},
	}

	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, manifestJSON, 0644))

	// Create a mock agent server
	server := &agentServer{}

	// Create push request
	req := &api.PushRequest{
		Path: tempDir,
	}

	// Execute push
	resp, err := server.Push(context.Background(), req)

	// Should not error and should report no new files (.pennsieve folder ignored)
	assert.NoError(t, err)
	assert.Contains(t, resp.Status, "No new files to push")
}

func TestPush_WithNestedFiles(t *testing.T) {
	// Create a temporary directory structure with manifest
	tempDir := t.TempDir()
	pennsieveDir := filepath.Join(tempDir, ".pennsieve")
	manifestPath := filepath.Join(pennsieveDir, "manifest.json")
	subDir := filepath.Join(tempDir, "subfolder")

	// Create directories
	require.NoError(t, os.MkdirAll(pennsieveDir, 0755))
	require.NoError(t, os.MkdirAll(subDir, 0755))

	// Create a new file in subfolder (not in manifest)
	newFile := filepath.Join(subDir, "nested.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("nested content"), 0644))

	// Create manifest with no files
	manifest := map[string]interface{}{
		"datasetNodeId":      "N:dataset:test-123",
		"organizationNodeId": "N:organization:test-456",
		"files":              []interface{}{},
	}

	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, manifestJSON, 0644))

	// Create a mock agent server
	server := &agentServer{}

	// Create push request
	req := &api.PushRequest{
		Path: tempDir,
	}

	// Execute push
	resp, err := server.Push(context.Background(), req)

	// Should not error and should find the nested file
	assert.NoError(t, err)
	assert.Contains(t, resp.Status, "Push initiated")
	assert.Contains(t, resp.Status, "1 file")
}
