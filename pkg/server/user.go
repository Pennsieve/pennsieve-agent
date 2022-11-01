package server

import (
	"context"
	"errors"
	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	log "github.com/sirupsen/logrus"
)

func (s *server) GetUser(ctx context.Context, request *pb.GetUserRequest) (*pb.UserResponse, error) {

	activeUser, err := s.User.GetActiveUser()
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

	s.stopSyncTimers()

	useConfig := s.client.GetAPIParams().UseConfigFile
	if !useConfig {
		return nil, errors.New("SWITCH is not available when agent is running without config file")
	}

	activeUser, err := s.User.SwitchUser(request.Profile)
	if err != nil {
		log.Error("Error:SwitchProfile: ", err)
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

	apiSession, _ := s.User.ReAuthenticate()
	activeUser, err := s.User.GetActiveUser()
	//activeUser, err := s.User.UpdateActiveUser()
	if err != nil {
		return nil, err
	}

	updatedUser, _ := s.User.UpdateTokenForUser(activeUser, &apiSession)

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
