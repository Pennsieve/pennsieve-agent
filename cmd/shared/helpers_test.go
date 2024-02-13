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

// Add test for helper functions as required
func (h *HelpersTestSuite) TestStubFunction() {
	h.Equal(true, true)
}

func TestDatasetsSuite(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
