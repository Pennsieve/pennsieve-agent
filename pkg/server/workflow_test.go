package server

import (
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
	
}

func TestWorkflowSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}
