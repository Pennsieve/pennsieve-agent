package server

import (
	"context"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	pb "github.com/pennsieve/pennsieve-agent/protos"
)

func (s *server) GetUser(ctx context.Context, request *pb.GetUserRequest) (*pb.UserResponse, error) {

	activeUser, err := api.GetActiveUser()
	if err != nil {
		return nil, err
	}

	resp := pb.UserResponse{
		Id:               activeUser.Id,
		Name:             activeUser.Name,
		SessionToken:     activeUser.SessionToken,
		Profile:          activeUser.Profile,
		Environment:      activeUser.Environment,
		OrganizationId:   activeUser.OrganizationId,
		OrganizationName: activeUser.OrganizationName,
	}
	return &resp, nil
}

func (s *server) SwitchProfile(ctx context.Context, request *pb.SwitchProfileRequest) (*pb.UserResponse, error) {

	activeUser, err := api.SwitchUser(request.Profile)
	if err != nil {
		return nil, err
	}

	resp := pb.UserResponse{
		Id:               activeUser.Id,
		Name:             activeUser.Name,
		SessionToken:     activeUser.SessionToken,
		Profile:          activeUser.Profile,
		Environment:      activeUser.Environment,
		OrganizationId:   activeUser.OrganizationId,
		OrganizationName: activeUser.OrganizationName,
	}
	return &resp, nil
}
