package server

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type UserTestSuite struct {
	ServerTestSuite
}

//func (s *UserTestSuite) TestGetUser() {
//req := v1.GetUserRequest{}
//resp, err := s.testServer.GetUser(context.Background(), &req)
//if s.NoError(err) {
//	expectedUserProfile := expectedUserProfiles[0]
//	s.Equal(expectedUserProfile.User.PreferredOrganization, resp.OrganizationId)
//	s.Equal(expectedUserProfile.User.ID, resp.Id)
//	s.Equal(fmt.Sprintf("%s %s",
//		expectedUserProfile.User.FirstName, expectedUserProfile.User.LastName), resp.Name)
//	s.Contains(s.mockPennsieve.JWTToAPIKey, resp.SessionToken)
//	s.Equal(expectedUserProfile.Profile.APIToken, s.mockPennsieve.JWTToAPIKey[resp.SessionToken])
//
//	s.Equal(expectedUserProfile.Profile.Name, resp.Profile)
//	s.Equal(expectedUserProfile.Org.ID, resp.OrganizationId)
//	s.Equal(expectedUserProfile.Org.Name, resp.OrganizationName)
//	s.Equal(s.mockPennsieve.API.Server.URL, resp.ApiHost)
//	s.NotEmpty(resp.Api2Host)
//}
//}

//
//func (s *UserTestSuite) TestSwitchProfile() {
//	switchReq := v1.SwitchProfileRequest{Profile: expectedUserProfiles[1].Profile.Name}
//	switchResp, err := s.testServer.SwitchProfile(context.Background(), &switchReq)
//	if s.NoError(err, "could not switch profile: %s", err) {
//		s.Equal(expectedUserProfiles[1].Profile.Name, switchResp.Profile)
//		s.Equal(expectedUserProfiles[1].Org.ID, switchResp.OrganizationId)
//		s.Equal(s.mockPennsieve.API.Server.URL, switchResp.ApiHost)
//		s.NotEmpty(switchResp.Api2Host)
//	}
//}

func TestUserSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}
