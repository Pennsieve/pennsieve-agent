package server

import (
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/manifest/manifestFile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"path/filepath"
	"testing"
)

type WorkflowTestSuite struct {
	server
	suite.Suite
}

var (
	singleRootDir *api.ListManifestFilesResponse
)

/*
getRootDirectories takes in *api.ListManifestFilesResponse
and returns only the root most directories as an array
*/
func (s *WorkflowTestSuite) TestGetRootDirectories() {
	expectedSingleRootDirResponse := []string{"/Users/pennUser/Documents/bids-data"}
	actualSingleRootDirResponse := getRootDirectories(singleRootDir)

	// Add a different root dir path
	multipleRootDirs := singleRootDir
	newFile := api.ListManifestFilesResponse_FileUpload{
		Id:         61,
		ManifestId: 10,
		SourcePath: filepath.Clean("/Volumes/mounted/.background/file3.md"),
		TargetPath: "",
		UploadId:   "7d64320a-6399-4671-b2a8-6eec4e2650c5",
		Status:     api.ListManifestFilesResponse_StatusType(manifestFile.Verified),
	}
	multipleRootDirs.File = append(multipleRootDirs.File, &newFile)

	expectedMultipleRootDirResponse := append(expectedSingleRootDirResponse, filepath.Clean("/Volumes/mounted/.background"))
	actualMultipleRootDirs := getRootDirectories(singleRootDir)

	assert.Equal(s.T(), expectedSingleRootDirResponse, actualSingleRootDirResponse)
	assert.Contains(s.T(), expectedMultipleRootDirResponse, actualMultipleRootDirs[0])
	assert.Contains(s.T(), expectedMultipleRootDirResponse, actualMultipleRootDirs[1])
}

func TestWorkflowSuite(t *testing.T) {
	singleRootDir = &api.ListManifestFilesResponse{
		File: []*api.ListManifestFilesResponse_FileUpload{
			{
				Id:         61,
				ManifestId: 10,
				SourcePath: filepath.Clean("/Users/pennUser/Documents/bids-data/.DS_Store"),
				TargetPath: "",
				UploadId:   "7d64320a-6399-4671-b2a8-6eec4e2650c1",
				Status:     api.ListManifestFilesResponse_StatusType(manifestFile.Verified),
			},
			{
				Id:         62,
				ManifestId: 10,
				SourcePath: filepath.Clean("/Users/pennUser/Documents/bids-data/.bidsignore"),
				TargetPath: "",
				UploadId:   "7d64320a-6399-4671-b2a8-6eec4e2650c2",
				Status:     api.ListManifestFilesResponse_StatusType(manifestFile.Verified),
			},
			{
				Id:         63,
				ManifestId: 10,
				SourcePath: filepath.Clean("/Users/pennUser/Documents/bids-data/sub-0001/ses-preimplant0001/eeg/sub-0001_ses-preimplant0001_task-task_run-01_eeg.json"),
				TargetPath: "",
				UploadId:   "7d64320a-6399-4671-b2a8-6eec4e2650c3",
				Status:     api.ListManifestFilesResponse_StatusType(manifestFile.Verified),
			},
			{
				Id:         64,
				ManifestId: 10,
				SourcePath: filepath.Clean("/Users/pennUser/Documents/bids-data/sub-0001/.DS_Store"),
				TargetPath: "",
				UploadId:   "7d64320a-6399-4671-b2a8-6eec4e2650c4",
				Status:     api.ListManifestFilesResponse_StatusType(manifestFile.Verified),
			},
		},
	}
	suite.Run(t, new(WorkflowTestSuite))
}
