package server

import (
	"context"
	v1 "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/stretchr/testify/suite"
	"testing"
)

type UserTestSuite struct {
	ServerTestSuite
}

func (s *UserTestSuite) TestSwitchProfile() {
	server, err := newServer(s.db,
		&pennsieve.AWSCognitoEndpoints{IdentityProviderEndpoint: s.mockPennsieve.IDProvider.Server.URL})
	if s.NoError(err) {
		switchReq := v1.SwitchProfileRequest{
			Profile: expectedUserProfiles[1].Profile.Name,
		}
		switchResp, err := server.SwitchProfile(context.Background(), &switchReq)
		if s.NoError(err, "could not switch profile: %s", err) {
			s.Equal(expectedUserProfiles[1].Profile.Name, switchResp.Profile)
			s.Equal(expectedUserProfiles[1].Org.ID, switchResp.OrganizationId)
			s.Equal(s.mockPennsieve.API.Server.URL, switchResp.ApiHost)
		}
	}

}

func TestUserSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}
