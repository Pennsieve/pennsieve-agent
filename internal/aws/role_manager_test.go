package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleManager(t *testing.T) {
	accountId := "someAccountId"
	profile := "someProfile"

	roleManager := NewAWSRoleManager(accountId, profile)

	_, err := roleManager.Create()
	assert.Error(t, err)
}
