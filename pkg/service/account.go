package service

import (
	"context"
	"fmt"

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

	roleName := fmt.Sprintf("ROLE-%s", pennsieveAccountId)
	registration := aws.NewAWSRoleManager(pennsieveAccountId, profile, roleName)

	// Get External AccountId
	externalAccountId, err := registration.GetAccountId()
	if err != nil {
		return nil, err
	}

	// registration
	_, err = registration.Create()
	if err != nil {
		return nil, err
	}

	_, err = a.PostAccounts(externalAccountId, accountType, roleName, "")
	if err != nil {
		return nil, err
	}

	return &api.RegisterResponse{AccountId: externalAccountId}, nil
}
