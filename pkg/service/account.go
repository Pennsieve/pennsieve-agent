package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"log"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/internal/aws"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
)

// roleConfig is the response from the account-service /role-policy endpoint.
type roleConfig struct {
	RoleName            string          `json:"roleName"`
	PolicyDocument      json.RawMessage `json:"policyDocument"`
	TrustPolicyDocument json.RawMessage `json:"trustPolicyDocument"`
}

type AccountService struct {
	Client *pennsieve.Client
}

func NewAccountService(client *pennsieve.Client) *AccountService {
	return &AccountService{Client: client}
}

func (a *AccountService) PostAccounts(accountId string, accountType string, roleName string, externalId string) (string, error) {
	resp, err := a.Client.Account.CreateAccount(context.Background(), accountId, accountType, roleName, externalId)
	if err != nil {
		return "", err
	}

	return resp.Uuid, nil
}

func (a *AccountService) fetchRoleConfig() (*roleConfig, error) {
	apiParams := a.Client.GetAPIParams()
	url := fmt.Sprintf("%s/compute/resources/role-policy", apiParams.ApiHost2)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating role-policy request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.Client.APISession.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching role-policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("role-policy endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading role-policy response: %w", err)
	}

	var config roleConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("parsing role-policy response: %w", err)
	}

	return &config, nil
}

func (a *AccountService) DeregisterAWS(profile string, accountType string, force bool) (*api.DeregisterResponse, error) {
	// 1. Get the external AWS account ID via STS
	tempManager := aws.NewAWSRoleManager(profile, "", "", "")
	externalAccountId, err := tempManager.GetAccountId()
	if err != nil {
		return nil, fmt.Errorf("getting AWS account ID: %w", err)
	}

	// 2. Get user's registered accounts
	accounts, err := a.Client.Account.GetAccounts(context.Background())
	if err != nil {
		return nil, fmt.Errorf("getting registered accounts: %w", err)
	}

	// 3. Find the account matching the external AWS account ID
	var matchedUuid string
	var matchCount int
	for _, acct := range accounts {
		if acct.AccountId == externalAccountId {
			matchedUuid = acct.Uuid
			matchCount++
		}
	}

	if matchCount == 0 {
		return nil, fmt.Errorf("no registered account found for AWS account %s", externalAccountId)
	}
	if matchCount > 1 {
		return nil, fmt.Errorf("multiple registered accounts found for AWS account %s; please contact support", externalAccountId)
	}

	// 4. Delete the Pennsieve account record
	deleteResp, err := a.Client.Account.DeleteAccount(context.Background(), matchedUuid, force)
func (a *AccountService) RequestEcrAccess(accountId string, accountType string) error {
	err := a.Client.Account.RequestEcrAccess(context.Background(), accountId, accountType)
	if err != nil {
		return err
	}
	return nil
}

func (a *AccountService) RegisterAWS(profile string, accountType string) (*api.RegisterResponse, error) {
	pennsieveAccountId, err := a.GetPennsieveAccounts(accountType)
	if err != nil {
		return nil, err
	}

	// 5. Delete the IAM role
	registration := aws.NewAWSRoleManager(profile, deleteResp.RoleName, "", "")
	if err := registration.Delete(); err != nil {
		// Pennsieve record is already deleted — warn but don't fail
		fmt.Printf("Warning: Pennsieve account record deleted, but failed to delete IAM role %s: %v\n", deleteResp.RoleName, err)
		fmt.Println("You may need to manually delete the IAM role from your AWS account.")
	}

	return &api.DeregisterResponse{
		AccountId: externalAccountId,
		RoleName:  deleteResp.RoleName,
	}, nil
}

func (a *AccountService) RegisterAWS(profile string, accountType string) (*api.RegisterResponse, error) {
	config, err := a.fetchRoleConfig()
	if err != nil {
		return nil, fmt.Errorf("fetching role config: %w", err)
	}

	roleName := config.RoleName
	trustPolicy := string(config.TrustPolicyDocument)
	permissionPolicy := string(config.PolicyDocument)
	registration := aws.NewAWSRoleManager(profile, roleName, trustPolicy, permissionPolicy)

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

	// Request ECR pull access for the newly registered account.
	if err := a.RequestEcrAccess(externalAccountId, accountType); err != nil {
		log.Printf("warning: failed to request ECR access for account %s: %v", externalAccountId, err)
	}

	return &api.RegisterResponse{AccountId: externalAccountId}, nil
}
