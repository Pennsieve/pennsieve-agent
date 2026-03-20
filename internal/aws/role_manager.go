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

	"github.com/pennsieve/pennsieve-agent/internal/account"
)

type AWSRoleManager struct {
	Profile          string
	RoleName         string
	TrustPolicy      string
	PermissionPolicy string
}

func NewAWSRoleManager(profile string, roleName string, trustPolicy string, permissionPolicy string) account.Registration {
	return &AWSRoleManager{Profile: profile, RoleName: roleName, TrustPolicy: trustPolicy, PermissionPolicy: permissionPolicy}
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

	// Clean up trust policy file
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

func (r *AWSRoleManager) Create() ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("error getting home directory:", err)
		return nil, err
	}
	pennsieveFolder := filepath.Join(home, ".pennsieve")

	// write trust policy (provided by account-service)
	trustPolicyFile := fmt.Sprintf("TRUST_POLICY_%s.json", r.RoleName)

	trustPolicyFileLocation := filepath.Join(pennsieveFolder, trustPolicyFile)
	err = os.WriteFile(trustPolicyFileLocation, []byte(r.TrustPolicy), 0644)
	if err != nil {
		log.Println("error writing trust policy data:", err)
		return nil, err
	}

	// create permission policy (keyed by RoleName to avoid races between concurrent registrations)
	permissionPolicyFile := fmt.Sprintf("PERMISSION_POLICY_%s.json", r.RoleName)

	permissionPolicyFileLocation := filepath.Join(pennsieveFolder, permissionPolicyFile)
	err = os.WriteFile(permissionPolicyFileLocation, []byte(r.PermissionPolicy), 0644)
	if err != nil {
		log.Println("error writing permission policy data:", err)
		return nil, err
	}

	// create role
	cmd := exec.Command("aws",
		"--profile", r.Profile,
		"iam", "create-role",
		"--role-name", r.RoleName,
		"--assume-role-policy-document", fmt.Sprintf("file://%s", trustPolicyFileLocation),
		"--max-session-duration", "7200")
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err = cmd.Run()
	if err != nil {
		if strings.Contains(errb.String(), "EntityAlreadyExists") {
			log.Println("role already exists")
			return nil, nil
		}
		return nil, errors.New(errb.String())
	}

	// attach inline permissions (keyed by RoleName for uniqueness)
	policyName := fmt.Sprintf("POLICY-%s", r.RoleName)
	permissionsCmd := exec.Command("aws",
		"--profile", r.Profile,
		"iam", "put-role-policy",
		"--policy-name", policyName,
		"--role-name", r.RoleName,
		"--policy-document", fmt.Sprintf("file://%s", permissionPolicyFileLocation))
	var poutb, perrb bytes.Buffer
	permissionsCmd.Stdout = &poutb
	permissionsCmd.Stderr = &perrb
	err = permissionsCmd.Run()
	if err != nil {
		log.Println(perrb.String())
		return nil, errors.New(perrb.String())
	}

	return outb.Bytes(), nil
}
