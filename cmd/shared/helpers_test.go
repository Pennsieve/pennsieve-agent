package shared

import (
	"github.com/stretchr/testify/suite"
)

import (
	"testing"
)

type HelpersTestSuite struct {
	suite.Suite
}

func (h *HelpersTestSuite) TestGetLeafDirectory() {
	nixDirs := [3]string{
		"/Users/etrakand/Documents/scan-data-01.02.03",
		"/Users/ralthor/Downloads/chop_data",
		"~/docs/folder",
	}

	winDirs := [3]string{
		"C:\\Documents\\pebara\\wolves",
	}
	h.Equal(GetLeafDirectory(nixDirs[0], "/"), "scan-data-01.02.03")
	h.Equal(GetLeafDirectory(nixDirs[1], "/"), "chop_data")
	h.Equal(GetLeafDirectory(nixDirs[2], "/"), "folder")

	h.Equal(GetLeafDirectory(winDirs[0], "\\"), "wolves")

}

func TestDatasetsSuite(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
