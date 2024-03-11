package server

import (
	"context"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/internal/aws"
)

func (s *server) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
	accountId := int64(1) // get accountId via accounts-service/accounts/pennsieve
	aws.CreateRole(accountId, req.Credentials.Profile)

	return &api.RegisterResponse{Id: accountId}, nil
}
