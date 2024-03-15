package server

import (
	"context"
	"fmt"
	"testing"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/stretchr/testify/suite"
)

type AccountTestSuite struct {
	ServerTestSuite
}

func (s *AccountTestSuite) TestRegister() {
	ctx := context.Background()

	awsAccountType := "Azure"
	value, ok := api.Account_AccountType_value[awsAccountType]
	if !ok {
		panic("invalid accountType value")
	}

	profile := "default"
	_, err := s.testServer.Register(ctx,
		&api.RegisterRequest{Account: &api.Account{Type: api.Account_AccountType(value)},
			Credentials: &api.Credentials{Profile: profile},
		})

	if s.Error(err) {
		s.Equal(fmt.Sprintf("unsupported accountType: %s", awsAccountType), err.Error())
	}
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
