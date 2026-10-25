package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/internal/aws"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
)

// roleConfig is a role setup from account-service (GET /role-policy): the
// role's name, the external ID Pennsieve issued for it (to this user, for 24
// hours), and how to create it from Pennsieve's CloudFormation template.
type roleConfig struct {
	RoleName           string          `json:"roleName"`
	PolicyDocument     json.RawMessage `json:"policyDocument"`
	ExternalId         string          `json:"externalId,omitempty"`
	PennsieveAccountId string          `json:"pennsieveAccountId,omitempty"`
	CloudFormation     *struct {
		TemplateUrl     string `json:"templateUrl"`
		TemplateVersion string `json:"templateVersion"`
	} `json:"cloudFormation,omitempty"`
}

// accountRecord is an account as account-service returns it to its owner,
// with how its role was made.
type accountRecord struct {
	Uuid                  string `json:"uuid"`
	AccountId             string `json:"accountId"`
	RoleName              string `json:"roleName"`
	ExternalId            string `json:"externalId"`
	Connection            string `json:"connection"`
	TemplateVersion       string `json:"templateVersion"`
	LatestTemplateVersion string `json:"latestTemplateVersion"`
}

const connectionCloudFormation = "cloudformation"

type AccountService struct {
	Client *pennsieve.Client
}

func NewAccountService(client *pennsieve.Client) *AccountService {
	return &AccountService{Client: client}
}

// apiCall sends a request to account-service with the user's session and
// decodes a 2xx answer into out (when not nil). A refusal comes back as its
// {"message": …}, which says why.
func (a *AccountService) apiCall(method, path string, body any, out any) error {
	apiParams := a.Client.GetAPIParams()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, apiParams.ApiHost2+"/compute/resources"+path, reader)
	if err != nil {
		return fmt.Errorf("creating %s request: %w", path, err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.Client.APISession.Token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling %s: %w", path, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading %s response: %w", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return refusal(resp.StatusCode, respBody)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("parsing %s response: %w", path, err)
	}
	return nil
}

// refusal is account-service's reason for a non-2xx answer: its "message",
// else the body, else the status.
func refusal(status int, body []byte) error {
	var m struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &m) == nil && m.Message != "" {
		return errors.New(m.Message)
	}
	if text := strings.TrimSpace(string(body)); text != "" {
		return fmt.Errorf("status %d: %s", status, text)
	}
	return fmt.Errorf("status %d", status)
}

// fetchRoleConfig gets a role setup: a new role name with an external ID
// issued for it, and the CloudFormation template to create it from.
func (a *AccountService) fetchRoleConfig() (*roleConfig, error) {
	var config roleConfig
	if err := a.apiCall("GET", "/role-policy?externalId=true", nil, &config); err != nil {
		return nil, fmt.Errorf("fetching the role setup: %w", err)
	}
	if config.CloudFormation == nil || config.CloudFormation.TemplateUrl == "" || config.ExternalId == "" || config.PennsieveAccountId == "" {
		return nil, errors.New("Pennsieve didn't return a CloudFormation template for the role; this Pennsieve can't connect accounts with this agent yet")
	}
	return &config, nil
}

// findAccount is the user's one account for externalAccountId.
func (a *AccountService) findAccount(externalAccountId string) (*accountRecord, error) {
	var accounts []accountRecord
	if err := a.apiCall("GET", "/accounts", nil, &accounts); err != nil {
		return nil, fmt.Errorf("getting registered accounts: %w", err)
	}
	var found []accountRecord
	for _, acct := range accounts {
		if acct.AccountId == externalAccountId {
			found = append(found, acct)
		}
	}
	switch len(found) {
	case 0:
		return nil, fmt.Errorf("no registered account found for AWS account %s", externalAccountId)
	case 1:
		return &found[0], nil
	default:
		return nil, fmt.Errorf("multiple registered accounts found for AWS account %s; please contact support", externalAccountId)
	}
}

// createRoleStack creates the role setup's role as a CloudFormation stack
// (named after the role) and waits for it.
func createRoleStack(stacks *aws.StackManager, config *roleConfig) error {
	if err := stacks.Create(config.RoleName, config.CloudFormation.TemplateUrl, config.RoleName, config.ExternalId, config.PennsieveAccountId); err != nil {
		return err
	}
	return stacks.Wait(config.RoleName, "CREATE_COMPLETE")
}

// DeregisterAWS removes the user's registration of the AWS account. A
// legacy role (made by an older agent) is deleted with it. A role from the
// CloudFormation template belongs to its stack, so stackName names it for the
// caller to offer deleting (deleting the role alone would leave the stack
// broken).
func (a *AccountService) DeregisterAWS(profile string, accountType string, force bool) (resp *api.DeregisterResponse, stackName string, err error) {
	// 1. Get the external AWS account ID via STS
	externalAccountId, err := aws.NewAWSRoleManager(profile, "").GetAccountId()
	if err != nil {
		return nil, "", fmt.Errorf("getting AWS account ID: %w", err)
	}

	// 2. The user's registration of it
	acct, err := a.findAccount(externalAccountId)
	if err != nil {
		return nil, "", err
	}

	// 3. Delete the Pennsieve account record
	deleteResp, err := a.Client.Account.DeleteAccount(context.Background(), acct.Uuid, force)
	if err != nil {
		return nil, "", fmt.Errorf("deleting account record: %w", err)
	}
	resp = &api.DeregisterResponse{AccountId: externalAccountId, RoleName: deleteResp.RoleName}

	// 4. The role
	stacks := aws.NewStackManager(profile)
	if acct.Connection == connectionCloudFormation || stacks.Exists(deleteResp.RoleName) {
		return resp, deleteResp.RoleName, nil
	}
	if err := aws.NewAWSRoleManager(profile, deleteResp.RoleName).Delete(); err != nil {
		// Pennsieve record is already deleted — warn but don't fail
		fmt.Printf("Warning: Pennsieve account record deleted, but failed to delete IAM role %s: %v\n", deleteResp.RoleName, err)
		fmt.Println("You may need to manually delete the IAM role from your AWS account.")
	}
	return resp, "", nil
}

func (a *AccountService) RequestEcrAccess(accountId string, accountType string) error {
	err := a.Client.Account.RequestEcrAccess(context.Background(), accountId, accountType)
	if err != nil {
		return err
	}
	return nil
}

// UpdateRoleAWS brings the account's role up to Pennsieve's latest template.
// A role from the template is a stack, updated in place (update-stack). A
// legacy role is replaced: a new stack from the template, then the account
// switches to it, and oldRoleName is the role for the user to delete
// (Pennsieve's own policy forbids it to delete Pennsieve-Compute-* roles).
func (a *AccountService) UpdateRoleAWS(profile string, accountType string) (resp *api.UpdateRoleResponse, oldRoleName string, err error) {
	externalAccountId, err := aws.NewAWSRoleManager(profile, "").GetAccountId()
	if err != nil {
		return nil, "", fmt.Errorf("getting AWS account ID: %w", err)
	}
	acct, err := a.findAccount(externalAccountId)
	if err != nil {
		return nil, "", err
	}
	// The template to update to. A role setup also issues a code, which, if
	// unused, simply expires.
	config, err := a.fetchRoleConfig()
	if err != nil {
		return nil, "", err
	}
	stacks := aws.NewStackManager(profile)

	if acct.Connection == connectionCloudFormation {
		updated, err := stacks.Update(acct.RoleName, config.CloudFormation.TemplateUrl)
		if err != nil {
			return nil, "", err
		}
		if updated {
			if err := stacks.Wait(acct.RoleName, "UPDATE_COMPLETE"); err != nil {
				return nil, "", err
			}
		}
		version := config.CloudFormation.TemplateVersion
		if version != "" && version != acct.TemplateVersion {
			if err := a.apiCall("PATCH", "/accounts/"+acct.Uuid, map[string]string{"templateVersion": version}, nil); err != nil {
				return nil, "", fmt.Errorf("recording template v%s for the account: %w", version, err)
			}
		}
		return &api.UpdateRoleResponse{AccountId: externalAccountId, RoleName: acct.RoleName}, "", nil
	}

	// A legacy role: a new one from the template, then the account uses it.
	if err := createRoleStack(stacks, config); err != nil {
		return nil, "", err
	}
	var switched struct {
		OldRoleName string `json:"oldRoleName"`
		RoleName    string `json:"roleName"`
	}
	body := map[string]string{"roleName": config.RoleName, "externalId": config.ExternalId, "templateVersion": config.CloudFormation.TemplateVersion}
	if err := a.apiCall("POST", "/accounts/"+acct.Uuid+"/role", body, &switched); err != nil {
		return nil, "", fmt.Errorf("switching the account to role %s (stack %s is in your account; delete it if you don't retry): %w", config.RoleName, config.RoleName, err)
	}
	return &api.UpdateRoleResponse{AccountId: externalAccountId, RoleName: config.RoleName}, switched.OldRoleName, nil
}

// RegisterAWS connects the AWS account the profile is for: it creates the
// role from Pennsieve's CloudFormation template (a stack named after the
// role), waits for it, and registers the account, which Pennsieve does only
// once it can use the role.
func (a *AccountService) RegisterAWS(profile string, accountType string) (*api.RegisterResponse, error) {
	config, err := a.fetchRoleConfig()
	if err != nil {
		return nil, err
	}

	externalAccountId, err := aws.NewAWSRoleManager(profile, "").GetAccountId()
	if err != nil {
		return nil, err
	}

	if err := createRoleStack(aws.NewStackManager(profile), config); err != nil {
		return nil, err
	}

	// Pennsieve checks it can use the role (and that the role requires its
	// external ID) before registering it; its refusal says why.
	var created struct {
		Uuid string `json:"uuid"`
	}
	body := map[string]string{
		"accountId":       externalAccountId,
		"accountType":     accountType,
		"roleName":        config.RoleName,
		"externalId":      config.ExternalId,
		"templateVersion": config.CloudFormation.TemplateVersion,
	}
	if err := a.apiCall("POST", "/accounts", body, &created); err != nil {
		return nil, fmt.Errorf("registering role %s in AWS account %s with Pennsieve (its stack %s is in your account; delete it if you don't retry): %w",
			config.RoleName, externalAccountId, config.RoleName, err)
	}

	// Request ECR pull access for the newly registered account.
	if err := a.RequestEcrAccess(externalAccountId, accountType); err != nil {
		log.Printf("warning: failed to request ECR access for account %s: %v", externalAccountId, err)
	}

	return &api.RegisterResponse{AccountId: externalAccountId}, nil
}
