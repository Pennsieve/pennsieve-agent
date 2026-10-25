package aws

import (
	"errors"
	"os"
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

// A policy document is on disk only while the aws CLI reads it, and only
// readable by the user.
func TestWithPolicyFile_RemovesTheFile(t *testing.T) {
	var path string
	err := withPolicyFile(`{"Version":"2012-10-17"}`, func(p string) error {
		path = p
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
		b, err := os.ReadFile(p)
		assert.Equal(t, `{"Version":"2012-10-17"}`, string(b))
		return err
	})
	assert.NoError(t, err)
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "the policy file should be gone")
}

func TestWithPolicyFile_RemovesTheFileOnError(t *testing.T) {
	var path string
	err := withPolicyFile("{}", func(p string) error {
		path = p
		return errors.New("aws failed")
	})
	assert.EqualError(t, err, "aws failed")
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))
}
