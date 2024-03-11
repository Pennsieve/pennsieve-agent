package server

import (
	"context"
	"fmt"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
)

func (s *UserTestSuite) TestRegister() {
	ctx := context.Background()

	response, err := s.testServer.Register(ctx, &api.RegisterRequest{})
	if s.NoError(err) {
		fmt.Println(response)
	}
}
