package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleCreator(t *testing.T) {
	accountId := "941165240011"
	profile := "ih-app-deploy"

	roleCreator := NewAWSRoleCreator(accountId, profile)

	_, err := roleCreator.Create()
	assert.Error(t, err)
}
