package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

		return &api.RegisterResponse{
			AccountId: extractAccountId(awsRole.Role.Arn)}, nil
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported accountType: %s", accountType))
	}

}

func extractAccountId(roleArn string) string {
	parts := strings.Split(roleArn, ":")

	if len(parts) < 6 {
		return ""
	}
	return parts[4]
}
