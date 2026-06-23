package server

import (
	"context"
	"fmt"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pkg/errors"
)

func (s *agentServer) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
	accountType := req.Account.Type.String()

	switch accountType {
	case "AWS":
		return s.AccountService().RegisterAWS(req.Credentials.Profile, accountType)
	default:
		return nil, errors.New(fmt.Sprintf("unsupported accountType: %s", accountType))
	}
}

func (s *agentServer) UpdateRole(ctx context.Context, req *api.UpdateRoleRequest) (*api.UpdateRoleResponse, error) {
	accountType := req.Account.Type.String()

	switch accountType {
	case "AWS":
		return s.AccountService().UpdateRoleAWS(req.Credentials.Profile, accountType)
	default:
		return nil, errors.New(fmt.Sprintf("unsupported accountType: %s", accountType))
	}
}

func (s *agentServer) Deregister(ctx context.Context, req *api.DeregisterRequest) (*api.DeregisterResponse, error) {
	accountType := req.Account.Type.String()

	switch accountType {
	case "AWS":
		return s.AccountService().DeregisterAWS(req.Credentials.Profile, accountType, req.Force)
	default:
		return nil, errors.New(fmt.Sprintf("unsupported accountType: %s", accountType))
	}
}
