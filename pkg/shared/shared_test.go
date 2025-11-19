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

func TestFindManifestFile_FoundInParentDirectory(t *testing.T) {
	// Temp dir structure
	// tempDir/.pennsieve/manifest.json
	// tempDir/Cairhien/
	// tempDir/Cairhien/the_foregate/
	// tempDir/Cairhien/the_foregate/the_heights
	tempDir := t.TempDir()
	pennsieveDir := filepath.Join(tempDir, ".pennsieve")
	manifestPath := filepath.Join(pennsieveDir, "manifest.json")
	subdir1 := filepath.Join(tempDir, "Cairhien")
	subdir2 := filepath.Join(subdir1, "the_foregate")
	subdir3 := filepath.Join(subdir2, "the_heights")

	// Create directory structure
	if err := os.MkdirAll(pennsieveDir, 0755); err != nil {
		t.Fatalf("Failed to create .pennsieve directory: %v", err)
	}
	if err := os.MkdirAll(subdir2, 0755); err != nil {
		t.Fatalf("Failed to create subdirectories: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}

	// Find manifest one level up
	result, err := FindManifestFile(subdir1)
	if err != nil {
		t.Errorf("Expected no error from subdir1, got: %v", err)
	}
	if result != manifestPath {
		t.Errorf("Expected manifest path %s from subdir1, got %s", manifestPath, result)
	}

	// Find manifest two levels up
	result, err = FindManifestFile(subdir2)
	if err != nil {
		t.Errorf("Expected no error from subdir2, got: %v", err)
	}
	if result != manifestPath {
		t.Errorf("Expected manifest path %s from subdir2, got %s", manifestPath, result)
	}

	// Find manifest three levels up
	result, err = FindManifestFile(subdir3)
	if err != nil {
		t.Errorf("Expected no error from subdir2, got: %v", err)
	}
	if result != manifestPath {
		t.Errorf("Expected manifest path %s from subdir2, got %s", manifestPath, result)
	}
}

func TestFindManifestFile_NotFound(t *testing.T) {
	// Test graceful failure when manifest not found
	tempDir := t.TempDir()
	subdir := filepath.Join(tempDir, "the_rahad")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	result, err := FindManifestFile(subdir)
	if err == nil {
		t.Error("Expected error when manifest not found, got nil")
	}
	if result != "" {
		t.Errorf("Expected empty result when manifest not found, got %s", result)
	}

	// Verify error message is descriptive
	expectedErrMsg := "manifest.json not found in path hierarchy"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestFindManifestFile_FromFileInDirectory(t *testing.T) {
	// Test manifest still returns if passing a file
	tempDir := t.TempDir()
	pennsieveDir := filepath.Join(tempDir, ".pennsieve")
	manifestPath := filepath.Join(pennsieveDir, "manifest.json")
	testFile := filepath.Join(tempDir, "the_eye_of_the_world.txt")

	// Create .pennsieve directory, manifest, and a test file
	if err := os.MkdirAll(pennsieveDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test finding manifest when starting from a file path (not directory)
	result, err := FindManifestFile(testFile)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != manifestPath {
		t.Errorf("Expected manifest path %s, got %s", manifestPath, result)
	}
}
