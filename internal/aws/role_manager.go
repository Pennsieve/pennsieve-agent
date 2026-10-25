package aws

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pennsieve/pennsieve-agent/v2/internal/account"
)

// AWSRoleManager finds the user's AWS account and removes a legacy compute
// role (one the agent made itself, before roles came from Pennsieve's
// CloudFormation template; StackManager makes and removes those).
type AWSRoleManager struct {
	Profile  string
	RoleName string
}

func NewAWSRoleManager(profile string, roleName string) account.Registration {
	return &AWSRoleManager{Profile: profile, RoleName: roleName}
}

func (r *AWSRoleManager) GetAccountId() (string, error) {
	cmd := exec.Command("aws",
		"--profile", r.Profile,
		"sts", "get-caller-identity",
		"--query", "Account",
		"--output", "text")
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return "", errors.New(errb.String())
	}

	return strings.TrimSpace(outb.String()), nil
}

func (r *AWSRoleManager) Delete() error {
	// Delete the inline policy first (keyed by RoleName to avoid collisions
	// when multiple accounts share the same AWS account)
	policyName := fmt.Sprintf("POLICY-%s", r.RoleName)
	cmd := exec.Command("aws",
		"--profile", r.Profile,
		"iam", "delete-role-policy",
		"--policy-name", policyName,
		"--role-name", r.RoleName)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		// If policy doesn't exist, that's fine - continue to delete the role
		if !strings.Contains(errb.String(), "NoSuchEntity") {
			return fmt.Errorf("error deleting role policy: %s", errb.String())
		}
	}

	// Delete the role
	deleteCmd := exec.Command("aws",
		"--profile", r.Profile,
		"iam", "delete-role",
		"--role-name", r.RoleName)
	var deleteErrb bytes.Buffer
	deleteCmd.Stderr = &deleteErrb
	err = deleteCmd.Run()
	if err != nil {
		if !strings.Contains(deleteErrb.String(), "NoSuchEntity") {
			return fmt.Errorf("error deleting role: %s", deleteErrb.String())
		}
	}

	// Remove the policy files earlier agents left for this role (this one
	// writes none).
	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("warning: could not get home directory to clean up policy files:", err)
		return nil
	}
	trustPolicyFile := filepath.Join(home, ".pennsieve", fmt.Sprintf("TRUST_POLICY_%s.json", r.RoleName))
	if err := os.Remove(trustPolicyFile); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: could not remove trust policy file %s: %v", trustPolicyFile, err)
	}
	permissionPolicyFile := filepath.Join(home, ".pennsieve", fmt.Sprintf("PERMISSION_POLICY_%s.json", r.RoleName))
	if err := os.Remove(permissionPolicyFile); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: could not remove permission policy file %s: %v", permissionPolicyFile, err)
	}

	return nil
}
