package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleManager(t *testing.T) {
	roleManager := NewAWSRoleManager("someProfile", "someRoleName")
	assert.NotNil(t, roleManager)
}
