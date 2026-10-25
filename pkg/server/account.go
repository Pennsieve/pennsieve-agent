package server

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/internal/account"
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
		resp, oldRoleName, err := s.AccountService().UpdateRoleAWS(req.Credentials.Profile, accountType)
		if err == nil && oldRoleName != "" {
			// The legacy role the account no longer uses, for the CLI to
			// tell the user to delete (no field for it in UpdateRoleResponse).
			_ = grpc.SetTrailer(ctx, metadata.Pairs(account.TrailerOldRoleName, oldRoleName))
		}
		return resp, err
	default:
		return nil, errors.New(fmt.Sprintf("unsupported accountType: %s", accountType))
	}
}

func (s *agentServer) Deregister(ctx context.Context, req *api.DeregisterRequest) (*api.DeregisterResponse, error) {
	accountType := req.Account.Type.String()

	switch accountType {
	case "AWS":
		resp, stackName, err := s.AccountService().DeregisterAWS(req.Credentials.Profile, accountType, req.Force)
		if err == nil && stackName != "" {
			// The role's CloudFormation stack, for the CLI to offer deleting.
			_ = grpc.SetTrailer(ctx, metadata.Pairs(account.TrailerRoleStack, stackName))
		}
		return resp, err
	default:
		return nil, errors.New(fmt.Sprintf("unsupported accountType: %s", accountType))
	}
}
