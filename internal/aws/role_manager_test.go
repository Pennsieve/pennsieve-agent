package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleManager(t *testing.T) {
	accountId := "someAccountId"
	profile := "someProfile"
	roleName := "someRoleName"

	roleManager := NewAWSRoleManager(accountId, profile, roleName)

	_, err := roleManager.Create()
	assert.Error(t, err)
}
