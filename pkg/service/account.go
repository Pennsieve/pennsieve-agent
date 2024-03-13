package service

import (
	"encoding/json"
	"strings"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/internal/account"
	"github.com/pennsieve/pennsieve-agent/internal/aws"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
)

type AccountService struct {
	Client      *pennsieve.Client
	RoleCreator account.Registration
}

func NewAccountService(client *pennsieve.Client) *AccountService {
	return &AccountService{Client: client}
}

func (a *AccountService) GetPennsieveAccount(accountType string) (int64, error) {
	// TODO: make a call to account-service to retrieve the Pennsieve AWS Account
	return int64(0), nil
}

func (a *AccountService) RegisterAWS(accountId int64, profile string) (*api.RegisterResponse, error) {
	registration := aws.NewAWSRoleCreator(accountId, profile)
	data, err := registration.Create()
	if err != nil {
		return nil, err
	}

	awsRole := aws.AWSRole{}
	err = json.Unmarshal(data, &awsRole)
	if err != nil {
		return nil, err
	}

	return &api.RegisterResponse{
		AccountId: extractAccountId(awsRole.Role.Arn)}, nil
}

func extractAccountId(roleArn string) string {
	parts := strings.Split(roleArn, ":")

	if len(parts) < 6 {
		return ""
	}
	return parts[4]
}
