package server

import (
	"context"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// The account RPCs stay in the API so older clients get a reason, not
// Unimplemented: connecting, updating and disconnecting AWS accounts moved to
// the Pennsieve app (its CloudFormation template makes the role only in the
// account being connected, which the agent's stacks no longer satisfy).
var errAccountsInApp = status.Error(codes.FailedPrecondition,
	"retired: AWS accounts are connected, updated and disconnected in the Pennsieve app (Your compute resources)")

func (s *agentServer) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
	return nil, errAccountsInApp
}

func (s *agentServer) UpdateRole(ctx context.Context, req *api.UpdateRoleRequest) (*api.UpdateRoleResponse, error) {
	return nil, errAccountsInApp
}

func (s *agentServer) Deregister(ctx context.Context, req *api.DeregisterRequest) (*api.DeregisterResponse, error) {
	return nil, errAccountsInApp
}
