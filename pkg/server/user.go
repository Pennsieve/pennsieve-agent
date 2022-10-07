package server

import (
	"context"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"log"
)

func (s *server) GetUser(ctx context.Context, request *pb.GetUserRequest) (*pb.UserResponse, error) {

	activeUser, err := api.UpdateActiveUser()
	if err != nil {
		return nil, err
	}

	resp := pb.UserResponse{
		Id:               activeUser.Id,
		Name:             activeUser.Name,
		SessionToken:     activeUser.SessionToken,
		TokenExpire:      activeUser.TokenExpire.Unix(),
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
		log.Println("Error:SwitchProfile: ", err)
		return nil, err
	}

	resp := pb.UserResponse{
		Id:               activeUser.Id,
		Name:             activeUser.Name,
		SessionToken:     activeUser.SessionToken,
		TokenExpire:      activeUser.TokenExpire.Unix(),
		Profile:          activeUser.Profile,
		Environment:      activeUser.Environment,
		OrganizationId:   activeUser.OrganizationId,
		OrganizationName: activeUser.OrganizationName,
	}
	return &resp, nil
}

func (s *server) ReAuthenticate(ctx context.Context, request *pb.ReAuthenticateRequest) (*pb.UserResponse, error) {

	apiSession, _ := api.ReAuthenticate()
	activeUser, err := api.UpdateActiveUser()
	if err != nil {
		return nil, err
	}

	updatedUser, _ := models.UpdateTokenForUser(activeUser, &apiSession)

	resp := pb.UserResponse{
		Id:               updatedUser.Id,
		Name:             updatedUser.Name,
		SessionToken:     updatedUser.SessionToken,
		TokenExpire:      updatedUser.TokenExpire.Unix(),
		Profile:          updatedUser.Profile,
		Environment:      updatedUser.Environment,
		OrganizationId:   updatedUser.OrganizationId,
		OrganizationName: updatedUser.OrganizationName,
	}
	return &resp, nil

}
