package store

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/manifest/manifestFile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestManifestFileStore(t *testing.T) {
	tests := []struct {
		scenario string
		fixture  Fixture
		testFunc func(t *testing.T, fixture *Fixture)
	}{
		{"remove from manifest: no manifest found", removeFromManifestFixture, testRemoveFromManifestNoManifest},
		{"remove from manifest: none found", removeFromManifestFixture, testRemoveFromManifestNoFilesUnderPrefix},
		{"remove from manifest: one found", removeFromManifestFixture, testRemoveFromManifestOneFileUnderPrefix},
		{"remove from manifest: multiple files under prefix", removeFromManifestFixture, testRemoveFromManifestMultipleFilesUnderPrefix},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			fixture := &tt.fixture
			fixture.Setup(t)

			t.Cleanup(func() {
				fixture.Teardown(t)
			})

			tt.testFunc(t, fixture)
		})
	}
}

var removeFromManifestFixture = Fixture{
	ManifestParams: ManifestParams{
		UserId:           uuid.NewString(),
		UserName:         uuid.NewString(),
		OrganizationId:   uuid.NewString(),
		OrganizationName: uuid.NewString(),
		DatasetId:        uuid.NewString(),
		DatasetName:      uuid.NewString(),
	},
	ManifestFiles: []ManifestFile{
		{
			SourcePath: "/home/user/folder1/local.txt",
			TargetPath: "folder1",
			TargetName: "local.txt",
			UploadId:   uuid.New(),
			Status:     manifestFile.Local,
		},
		{
			SourcePath: "/home/user/folder1/registered.txt",
			TargetPath: "folder1",
			TargetName: "registered.txt",
			UploadId:   uuid.New(),
			Status:     manifestFile.Registered,
		},
		{
			SourcePath: "/home/user/folder1/finalized.txt",
			TargetPath: "folder1",
			TargetName: "finalized.txt",
			UploadId:   uuid.New(),
			Status:     manifestFile.Finalized,
		},
		{
			SourcePath: "/home/user/folder2/local1.txt",
			TargetPath: "folder2",
			TargetName: "local1.txt",
			UploadId:   uuid.New(),
			Status:     manifestFile.Local,
		},
		{
			SourcePath: "/home/user/folder2/local2.txt",
			TargetPath: "folder2",
			TargetName: "local2.txt",
			UploadId:   uuid.New(),
			Status:     manifestFile.Local,
		},
	},
}

func testRemoveFromManifestNoManifest(t *testing.T, fixture *Fixture) {
	resp, err := fixture.ManifestFileStore.RemoveFromManifest(fixture.Manifest.Id+1000, "/path/to/remove")
	require.NoError(t, err)

	assert.Equal(t, resp.Deleted, int64(0))
	assert.Equal(t, resp.Updated, int64(0))

	var expectedSourcePaths []string
	for _, file := range fixture.ManifestFiles {
		expectedSourcePaths = append(expectedSourcePaths, file.SourcePath)
	}

	assertions := []ManifestFilesAssertion{ManifestFilesLenAssertion(len(fixture.ManifestFiles))}
	for _, file := range fixture.ManifestFiles {
		assertions = append(assertions, ManifestFilesContainsSourcePathAndStatusAssertion(file.SourcePath, file.Status))
	}

	// all files should still be in the manifest with the same status
	fixture.AssertManifestFiles(t, assertions...)
}

func testRemoveFromManifestNoFilesUnderPrefix(t *testing.T, fixture *Fixture) {
	resp, err := fixture.ManifestFileStore.RemoveFromManifest(fixture.Manifest.Id, "/home/user/folder3/")
	require.NoError(t, err)

	assert.Equal(t, resp.Deleted, int64(0))
	assert.Equal(t, resp.Updated, int64(0))

	assertions := []ManifestFilesAssertion{ManifestFilesLenAssertion(len(fixture.ManifestFiles))}
	for _, file := range fixture.ManifestFiles {
		assertions = append(assertions, ManifestFilesContainsSourcePathAndStatusAssertion(file.SourcePath, file.Status))
	}

	// all files should still be in the manifest with the same status
	fixture.AssertManifestFiles(t, assertions...)
}

func testRemoveFromManifestOneFileUnderPrefix(t *testing.T, fixture *Fixture) {
	prefixToRemove := "/home/user/folder1/local.txt"
	resp, err := fixture.ManifestFileStore.RemoveFromManifest(fixture.Manifest.Id, prefixToRemove)
	require.NoError(t, err)

	assert.Equal(t, resp.Deleted, int64(1))
	assert.Equal(t, resp.Updated, int64(0))

	assertions := []ManifestFilesAssertion{
		// all but one file should remain
		ManifestFilesLenAssertion(len(fixture.ManifestFiles) - 1),
		// remaining files should be unchanged
		ManifestFilesContainsSourcePathAndStatusAssertion("/home/user/folder1/registered.txt", manifestFile.Registered),
		ManifestFilesContainsSourcePathAndStatusAssertion("/home/user/folder1/finalized.txt", manifestFile.Finalized),
		ManifestFilesContainsSourcePathAndStatusAssertion("/home/user/folder2/local1.txt", manifestFile.Local),
		ManifestFilesContainsSourcePathAndStatusAssertion("/home/user/folder2/local2.txt", manifestFile.Local),
	}

	fixture.AssertManifestFiles(t, assertions...)

}

func testRemoveFromManifestMultipleFilesUnderPrefix(t *testing.T, fixture *Fixture) {
	prefixToRemove := "/home/user/folder1/"
	resp, err := fixture.ManifestFileStore.RemoveFromManifest(fixture.Manifest.Id, prefixToRemove)
	require.NoError(t, err)

	assert.Equal(t, resp.Deleted, int64(1))
	assert.Equal(t, resp.Updated, int64(1))

	assertions := []ManifestFilesAssertion{
		// all but one file should remain
		ManifestFilesLenAssertion(len(fixture.ManifestFiles) - 1),
		// registered file with matching source path should be changed to Removed
		ManifestFilesContainsSourcePathAndStatusAssertion("/home/user/folder1/registered.txt", manifestFile.Removed),
		// remaining files should be unchanged
		ManifestFilesContainsSourcePathAndStatusAssertion("/home/user/folder1/finalized.txt", manifestFile.Finalized),
		ManifestFilesContainsSourcePathAndStatusAssertion("/home/user/folder2/local1.txt", manifestFile.Local),
		ManifestFilesContainsSourcePathAndStatusAssertion("/home/user/folder2/local2.txt", manifestFile.Local),
	}

	fixture.AssertManifestFiles(t, assertions...)

}

type Fixture struct {
	// Stores
	ManifestStore     *manifestStore
	ManifestFileStore *manifestFileStore

	//Inputs
	ManifestParams ManifestParams

	//Input and Output
	ManifestFiles []ManifestFile

	//Outputs
	Manifest *Manifest
}

func (f *Fixture) Setup(t *testing.T) {
	f.ManifestStore = NewManifestStore(db)
	f.ManifestFileStore = NewManifestFileStore(db)

	manifest, err := f.ManifestStore.Add(f.ManifestParams)
	require.NoError(t, err)
	f.Manifest = manifest

	f.addManifestFiles(t, f.Manifest.Id)

}

func (f *Fixture) addManifestFiles(t *testing.T, manifestID int32) {
	if len(f.ManifestFiles) == 0 {
		return
	}
	var uploadIDToFile = make(map[string]*ManifestFile, len(f.ManifestFiles))
	currentTime := time.Now()
	const rowSQL = "(?, ?, ?, ?, ?, ?, ?, ?)"
	var vals []interface{}
	var inserts []string

	sqlInsert := `INSERT INTO manifest_files(source_path, target_path, target_name, upload_id, manifest_id, status, created_at, updated_at) VALUES `
	for i := range f.ManifestFiles {
		file := &f.ManifestFiles[i]
		file.ManifestId = manifestID
		uploadID := file.UploadId.String()
		uploadIDToFile[uploadID] = file
		inserts = append(inserts, rowSQL)
		createdAt, updatedAt := file.CreatedAt, file.UpdatedAt
		if createdAt.IsZero() {
			createdAt = currentTime
		}
		if updatedAt.IsZero() {
			updatedAt = currentTime
		}
		vals = append(vals, file.SourcePath, file.TargetPath, file.TargetName, uploadID, file.ManifestId,
			file.Status.String(), createdAt, updatedAt)
	}
	fullInsert := fmt.Sprintf("%s %s RETURNING upload_id, id", sqlInsert, strings.Join(inserts, ", "))

	stmt, err := db.Prepare(fullInsert)
	require.NoError(t, err)
	defer func() { assert.NoError(t, stmt.Close()) }()

	rows, err := stmt.Query(vals...)
	require.NoError(t, err)
	defer func() { assert.NoError(t, rows.Close()) }()

	//SQLite makes no guarantee about the order of the returned rows so can't assume first id goes with first manifest file.
	// https://www.sqlite.org/lang_returning.html
	for rows.Next() {
		var id int32
		var uploadID string
		require.NoError(t, rows.Scan(&uploadID, &id))
		file, found := uploadIDToFile[uploadID]
		require.True(t, found)
		file.Id = id
	}

}

func (f *Fixture) Teardown(t *testing.T) {
	require.NoError(t, f.ManifestStore.Remove(f.Manifest.Id))
}

type ManifestFilesAssertion func(t *testing.T, actualManifestFiles []ManifestFile)

func ManifestFilesLenAssertion(expectedLen int) ManifestFilesAssertion {
	return func(t *testing.T, actualManifestFiles []ManifestFile) {
		assert.Len(t, actualManifestFiles, expectedLen, "expected %d files, found %d: %+v", expectedLen, len(actualManifestFiles), actualManifestFiles)
	}
}

func ManifestFilesContainsSourcePathAndStatusAssertion(expectedSourcePath string, expectedStatus manifestFile.Status) ManifestFilesAssertion {
	return func(t *testing.T, actualManifestFiles []ManifestFile) {
		actualIndex := slices.IndexFunc(actualManifestFiles, func(file ManifestFile) bool {
			return file.SourcePath == expectedSourcePath && file.Status == expectedStatus
		})
		require.True(t, actualIndex > -1, "no file with expected source path %s and status %s found", expectedSourcePath, expectedStatus.String())
	}
}

// AssertManifestFiles runs the given assertions over all the files that exist under the Fixture's manifest in the DB
// Assumes no manifest files were added except Fixture.ManifestFiles.
func (f *Fixture) AssertManifestFiles(t *testing.T, assertions ...ManifestFilesAssertion) {
	files, err := f.ManifestFileStore.Get(f.Manifest.Id, int32(len(f.ManifestFiles)), 0)
	require.NoError(t, err)

	for _, assertion := range assertions {
		assertion(t, files)
	}
}
