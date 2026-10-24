package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"log"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/internal/aws"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
)

// roleConfig is the response from the account-service /role-policy endpoint.
type roleConfig struct {
	RoleName            string          `json:"roleName"`
	PolicyDocument      json.RawMessage `json:"policyDocument"`
	TrustPolicyDocument json.RawMessage `json:"trustPolicyDocument"`
	// ExternalId is the sts:ExternalId the trust policy requires, issued when
	// asked for; empty from an account-service that doesn't issue them yet.
	ExternalId string `json:"externalId,omitempty"`
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

// fetchRoleConfig gets the role name and policies for a new role, with an
// external ID issued for it when withExternalId.
func (a *AccountService) fetchRoleConfig(withExternalId bool) (*roleConfig, error) {
	apiParams := a.Client.GetAPIParams()
	url := fmt.Sprintf("%s/compute/resources/role-policy", apiParams.ApiHost2)
	if withExternalId {
		url += "?externalId=true"
	}

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
	if err != nil {
		return nil, fmt.Errorf("deleting account record: %w", err)
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

func (a *AccountService) RequestEcrAccess(accountId string, accountType string) error {
	err := a.Client.Account.RequestEcrAccess(context.Background(), accountId, accountType)
	if err != nil {
		return err
	}
	return nil
}

func (a *AccountService) UpdateRoleAWS(profile string, accountType string) (*api.UpdateRoleResponse, error) {
	// 1. Resolve the external AWS account ID via STS.
	tempManager := aws.NewAWSRoleManager(profile, "", "", "")
	externalAccountId, err := tempManager.GetAccountId()
	if err != nil {
		return nil, fmt.Errorf("getting AWS account ID: %w", err)
	}

	// 2. Find the registered account so we can target its EXISTING role.
	//    fetchRoleConfig mints a fresh random role name on every call, so we
	//    must not use that name here — we reuse the one already on record.
	accounts, err := a.Client.Account.GetAccounts(context.Background())
	if err != nil {
		return nil, fmt.Errorf("getting registered accounts: %w", err)
	}

	var roleName, externalId string
	var matchCount int
	for _, acct := range accounts {
		if acct.AccountId == externalAccountId {
			roleName = acct.RoleName
			externalId = acct.ExternalId
			matchCount++
		}
	}
	if matchCount == 0 {
		return nil, fmt.Errorf("no registered account found for AWS account %s", externalAccountId)
	}
	if matchCount > 1 {
		return nil, fmt.Errorf("multiple registered accounts found for AWS account %s; please contact support", externalAccountId)
	}

	// 3. Fetch the latest policy documents and apply them to the existing role.
	//    The trust policy keeps the account's external ID condition: a fresh
	//    one would be issued for a new role, not this one.
	config, err := a.fetchRoleConfig(false)
	if err != nil {
		return nil, fmt.Errorf("fetching role config: %w", err)
	}
	trustPolicy := string(config.TrustPolicyDocument)
	if externalId != "" {
		trustPolicy, err = requireExternalId(trustPolicy, externalId)
		if err != nil {
			return nil, fmt.Errorf("building trust policy: %w", err)
		}
	}

	registration := aws.NewAWSRoleManager(profile, roleName, trustPolicy, string(config.PolicyDocument))
	if err := registration.Update(); err != nil {
		return nil, fmt.Errorf("updating role %s: %w", roleName, err)
	}

	return &api.UpdateRoleResponse{AccountId: externalAccountId, RoleName: roleName}, nil
}

func (a *AccountService) RegisterAWS(profile string, accountType string) (*api.RegisterResponse, error) {
	config, err := a.fetchRoleConfig(true)
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

	// Pennsieve checks it can use the role (and, with an external ID, that
	// the role requires it) before registering it; its refusal says why.
	_, err = a.PostAccounts(externalAccountId, accountType, roleName, config.ExternalId)
	if err != nil {
		return nil, fmt.Errorf("registering role %s in AWS account %s with Pennsieve: %w", roleName, externalAccountId, err)
	}

	// Request ECR pull access for the newly registered account.
	if err := a.RequestEcrAccess(externalAccountId, accountType); err != nil {
		log.Printf("warning: failed to request ECR access for account %s: %v", externalAccountId, err)
	}

	return &api.RegisterResponse{AccountId: externalAccountId}, nil
}

// requireExternalId adds an sts:ExternalId condition to every statement of a
// trust policy, so an updated role still requires its account's external ID.
func requireExternalId(trustPolicy string, externalId string) (string, error) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(trustPolicy), &doc); err != nil {
		return "", err
	}
	statements, ok := doc["Statement"].([]any)
	if !ok {
		return "", fmt.Errorf("trust policy has no statements")
	}
	for _, st := range statements {
		m, ok := st.(map[string]any)
		if !ok {
			return "", fmt.Errorf("unexpected trust policy statement")
		}
		m["Condition"] = map[string]any{
			"StringEquals": map[string]any{"sts:ExternalId": externalId},
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
