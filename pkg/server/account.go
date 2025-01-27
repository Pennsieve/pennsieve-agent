package server

import (
	"context"
	"fmt"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
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
