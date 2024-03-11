package server

import (
	"context"
	"fmt"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
)

func (s *UserTestSuite) TestRegister() {
	ctx := context.Background()

	awsAccountType := "AWS"
	value, ok := api.Account_AccountType_value[awsAccountType]
	if !ok {
		panic("invalid accountType value")
	}

	profile := "default"
	response, err := s.testServer.Register(ctx,
		&api.RegisterRequest{Account: &api.Account{Type: api.Account_AccountType(value)},
			Credentials: &api.Credentials{Profile: profile},
		})

	if s.NoError(err) {
		fmt.Println(response)
	}
}
