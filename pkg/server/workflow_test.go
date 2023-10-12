package server

import (
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type WorkflowTestSuite struct {
	server
	suite.Suite
}

func (s *WorkflowTestSuite) TestWorkflow() {

	filePaths := []string{
		"/Users/code/pennsieve-agent/File1.jpg",
		"/Users/code/pennsieve-agent/File2.jpg",
		"/Users/code/pennsieve-agent/File3.jpg",
		"/Users/code/pennsieve-agent/sub-folder/File3.jpg",
		"/Users/code/pennsieve-agent/sub-folder/sub-sub-folder/File3.jpg",
		"/Users/Documents/Adobe/File1.txt",
		"/Users/Documents/Adobe/File2.txt",
		"/Users/Documents/Adobe/images/File2.png",
		"/Users/Documents/Adobe/backup/File2.txt",
		"/Volumes/DBeaver Community/.background/file1.md",
		"/Volumes/DBeaver Community/.background/file3.md",
	}

	paths := commonPathParts(filePaths[0], filePaths[1])
	assert.Equal(s.T(), []string{"", "Users", "code", "pennsieve-agent"}, paths)

	paths = commonPathParts(filePaths[3], filePaths[4])
	assert.Equal(s.T(), []string{"", "Users", "code", "pennsieve-agent", "sub-folder"}, paths)

	responseFiles := []*api.ListManifestFilesResponse_FileUpload{
		{
			Id:         2,
			ManifestId: 2,
			SourcePath: filePaths[0],
			TargetPath: filePaths[0],
			UploadId:   "",
			Status:     0,
		},
		{
			Id:         3,
			ManifestId: 2,
			SourcePath: filePaths[1],
			TargetPath: filePaths[1],
			UploadId:   "",
			Status:     0,
		},
		{
			Id:         4,
			ManifestId: 2,
			SourcePath: filePaths[2],
			TargetPath: filePaths[2],
			UploadId:   "",
			Status:     0,
		},
		{
			Id:         5,
			ManifestId: 2,
			SourcePath: filePaths[3],
			TargetPath: filePaths[3],
			UploadId:   "",
			Status:     0,
		},
		{
			Id:         6,
			ManifestId: 2,
			SourcePath: filePaths[4],
			TargetPath: filePaths[4],
			UploadId:   "",
			Status:     0,
		},
		{
			Id:         7,
			ManifestId: 2,
			SourcePath: filePaths[5],
			TargetPath: filePaths[5],
			UploadId:   "",
			Status:     0,
		},
		{
			Id:         8,
			ManifestId: 2,
			SourcePath: filePaths[6],
			TargetPath: filePaths[6],
			UploadId:   "",
			Status:     0,
		},
		{
			Id:         9,
			ManifestId: 2,
			SourcePath: filePaths[7],
			TargetPath: filePaths[7],
			UploadId:   "",
			Status:     0,
		},
		{
			Id:         10,
			ManifestId: 2,
			SourcePath: filePaths[8],
			TargetPath: filePaths[8],
			UploadId:   "",
			Status:     0,
		},
		{
			Id:         11,
			ManifestId: 2,
			SourcePath: filePaths[9],
			TargetPath: filePaths[9],
			UploadId:   "",
			Status:     0,
		},
		{
			Id:         12,
			ManifestId: 2,
			SourcePath: filePaths[10],
			TargetPath: filePaths[10],
			UploadId:   "",
			Status:     0,
		},
	}
	response := api.ListManifestFilesResponse{File: responseFiles}

	rootDirs := getRootDirectories(&response)
	assert.Equal(s.T(), []string{"/Users/code/pennsieve-agent", "/Users/Documents/Adobe", "/Volumes/DBeaver Community/.background"}, rootDirs)

}

func TestWorkflowSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}
