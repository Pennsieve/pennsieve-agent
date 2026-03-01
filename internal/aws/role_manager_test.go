package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleManager(t *testing.T) {
	accountId := "someAccountId"
	profile := "someProfile"
	roleName := "someRoleName"
	permissionPolicy := "somePermissionPolicy"

	roleManager := NewAWSRoleManager(accountId, profile, roleName, permissionPolicy)

	_, err := roleManager.Create()
	assert.Error(t, err)
}
