package server

import (
	"context"
	"encoding/json"
	"fmt"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/internal/aws"
	"github.com/pkg/errors"
)

func (s *server) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
	accountType := req.Account.Type.String()
	switch accountType {
	case "AWS":
		accountId := 1 // TODO: get from account-service

		registration := aws.NewAWSRoleCreator(int64(accountId), req.Credentials.Profile)
		data, err := registration.Create()
		if err != nil {
			return nil, err
		}

		awsRole := aws.AWSRole{}
		err = json.Unmarshal(data, &awsRole)
		if err != nil {
			return nil, err
		}
		fmt.Println(awsRole)

		return &api.RegisterResponse{Id: 1}, nil
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported accountType: %s", accountType))
	}

}
