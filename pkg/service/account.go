package service

import (
	"context"
	"encoding/json"
	"strings"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/internal/aws"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
)

type AccountService struct {
	Client *pennsieve.Client
}

func NewAccountService(client *pennsieve.Client) *AccountService {
	return &AccountService{Client: client}
}

func (a *AccountService) GetPennsieveAccounts(accountType string) (string, error) {
	resp, err := a.Client.Account.GetPennsieveAccounts(context.Background(), accountType)
	if err != nil {
		return "", err
	}

	return resp.AccountId, nil
}

func (a *AccountService) PostAccounts(accountId string, accountType string, roleName string, externalId string) (string, error) {
	resp, err := a.Client.Account.CreateAccount(context.Background(), accountId, accountType, roleName, externalId)
	if err != nil {
		return "", err
	}

	return resp.Uuid, nil
}

func (a *AccountService) RegisterAWS(profile string, accountType string) (*api.RegisterResponse, error) {
	pennsieveAccountId, err := a.GetPennsieveAccounts(accountType)
	if err != nil {
		return nil, err
	}

	//registration
	registration := aws.NewAWSRoleManager(pennsieveAccountId, profile)
	data, err := registration.Create()
	if err != nil {
		return nil, err
	}

	awsRole := aws.AWSRole{}
	err = json.Unmarshal(data, &awsRole)
	if err != nil {
		return nil, err
	}

	externalAccountId := extractAccountId(awsRole.Role.Arn)
	_, err = a.PostAccounts(externalAccountId, accountType, awsRole.Role.RoleName, "")
	if err != nil {
		return nil, err
	}

	return &api.RegisterResponse{AccountId: externalAccountId}, nil
}

func extractAccountId(roleArn string) string {
	parts := strings.Split(roleArn, ":")

	if len(parts) < 6 {
		return ""
	}
	return parts[4]
}
