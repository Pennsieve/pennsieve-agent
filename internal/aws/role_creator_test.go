package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleCreator(t *testing.T) {
	accountId := "someAccountId"
	profile := "someProfile"

	roleCreator := NewAWSRoleCreator(accountId, profile)

	_, err := roleCreator.Create()
	assert.Error(t, err)
}
