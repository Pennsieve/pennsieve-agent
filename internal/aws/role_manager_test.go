package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleManager(t *testing.T) {
	profile := "someProfile"
	roleName := "someRoleName"
	trustPolicy := "someTrustPolicy"
	permissionPolicy := "somePermissionPolicy"

	roleManager := NewAWSRoleManager(profile, roleName, trustPolicy, permissionPolicy)

	_, err := roleManager.Create()
	assert.Error(t, err)
}
