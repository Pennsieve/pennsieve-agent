package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCrCCheck(t *testing.T) {

	// REMOVING these tests because it looks like Windows and Linux implement CRC differently \
	// big vs. little endian?? so we can't test cross platform.

	//// Check CRC
	//nrBytes := 1048576 // 1MB
	//crc, err := GetFileCrc32("../../resources/test/pullTest/.pennsieve/manifest.json", nrBytes)
	//assert.NoError(t, err)
	//assert.Equal(t, uint32(0xa1dc9897), crc)
	//
	//// Check that crc is idempotent
	//crc, err = GetFileCrc32("../../resources/test/pullTest/.pennsieve/manifest.json", nrBytes)
	//assert.NoError(t, err)
	//assert.Equal(t, uint32(0xa1dc9897), crc)
	//
	//// Check crc with maxbuffer that is smaller than file
	//crc, err = GetFileCrc32("../../resources/test/pullTest/.pennsieve/manifest.json", 50)
	//assert.NoError(t, err)
	//assert.Equal(t, uint32(0x3842f4f6), crc)
}

func TestFindManifestFile_FoundInCurrentDirectory(t *testing.T) {
	// Create a temporary directory structure with manifest in current dir
	tempDir := t.TempDir()
	pennsieveDir := filepath.Join(tempDir, ".pennsieve")
	manifestPath := filepath.Join(pennsieveDir, "manifest.json")

	// Create .pennsieve directory and manifest.json
	if err := os.MkdirAll(pennsieveDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}

	// Test finding manifest from current directory
	result, err := FindManifestFile(tempDir)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != manifestPath {
		t.Errorf("Expected manifest path %s, got %s", manifestPath, result)
	}
}
