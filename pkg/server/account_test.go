package server

import (
	"context"
	"testing"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AccountTestSuite struct {
	ServerTestSuite
}

// The retired account RPCs refuse with a reason pointing to the app, for any
// account type, without touching AWS.
func (s *AccountTestSuite) TestRetired() {
	ctx := context.Background()
	account := &api.Account{Type: api.Account_AccountType(api.Account_AccountType_value["AWS"])}
	creds := &api.Credentials{Profile: "default"}

	_, regErr := s.testServer.Register(ctx, &api.RegisterRequest{Account: account, Credentials: creds})
	_, updErr := s.testServer.UpdateRole(ctx, &api.UpdateRoleRequest{Account: account, Credentials: creds})
	_, deregErr := s.testServer.Deregister(ctx, &api.DeregisterRequest{Account: account, Credentials: creds})

	for _, err := range []error{regErr, updErr, deregErr} {
		st, ok := status.FromError(err)
		s.True(ok)
		s.Equal(codes.FailedPrecondition, st.Code())
		s.Contains(st.Message(), "Pennsieve app")
	}
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
