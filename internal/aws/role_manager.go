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

// withPolicyFile writes doc to a private temporary file for the length of fn
// (the aws CLI reads policy documents from file://), then removes it. The
// trust policy carries the account's external ID, so no policy document is
// left on disk.
func withPolicyFile(doc string, fn func(path string) error) error {
	f, err := os.CreateTemp("", "pennsieve-policy-*.json")
	if err != nil {
		return fmt.Errorf("creating policy file: %w", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(doc); err != nil {
		f.Close()
		return fmt.Errorf("writing policy file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("writing policy file: %w", err)
	}
	return fn(f.Name())
}

func (r *AWSRoleManager) Create() ([]byte, error) {
	// create role, with the trust policy provided by account-service
	var outb bytes.Buffer
	err := withPolicyFile(r.TrustPolicy, func(trustPolicyFile string) error {
		cmd := exec.Command("aws",
			"--profile", r.Profile,
			"iam", "create-role",
			"--role-name", r.RoleName,
			"--assume-role-policy-document", "file://"+trustPolicyFile,
			"--max-session-duration", "7200")
		var errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb
		if err := cmd.Run(); err != nil {
			// If the role already exists we still (re)attach the inline policy
			// below, so a re-run repairs a role whose policy is missing or stale.
			if !strings.Contains(errb.String(), "EntityAlreadyExists") {
				return errors.New(errb.String())
			}
			log.Println("role already exists; syncing inline policy")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// attach inline permissions (keyed by RoleName for uniqueness)
	if err := r.putRolePolicy(); err != nil {
		return nil, err
	}

	return outb.Bytes(), nil
}

// Update re-syncs an existing role's inline permission policy (and trust
// policy, when provided) with the latest documents from account-service.
// Unlike Create it does not create the role — put-role-policy is an upsert,
// so this safely overwrites the policy on a role that already exists without
// the disruption of deleting and recreating it.
func (r *AWSRoleManager) Update() error {
	if err := r.putRolePolicy(); err != nil {
		return err
	}

	// Refresh the trust policy too when one was supplied, so the role stays
	// in sync with what account-service currently serves.
	if r.TrustPolicy == "" {
		return nil
	}
	return withPolicyFile(r.TrustPolicy, func(trustPolicyFile string) error {
		trustCmd := exec.Command("aws",
			"--profile", r.Profile,
			"iam", "update-assume-role-policy",
			"--role-name", r.RoleName,
			"--policy-document", "file://"+trustPolicyFile)
		var terrb bytes.Buffer
		trustCmd.Stderr = &terrb
		if err := trustCmd.Run(); err != nil {
			log.Println(terrb.String())
			return errors.New(terrb.String())
		}
		return nil
	})
}

// putRolePolicy attaches (or overwrites) the inline permission policy on the
// role. The policy name is keyed by RoleName for uniqueness across accounts
// that share an AWS account.
func (r *AWSRoleManager) putRolePolicy() error {
	policyName := fmt.Sprintf("POLICY-%s", r.RoleName)
	return withPolicyFile(r.PermissionPolicy, func(permissionPolicyFile string) error {
		permissionsCmd := exec.Command("aws",
			"--profile", r.Profile,
			"iam", "put-role-policy",
			"--policy-name", policyName,
			"--role-name", r.RoleName,
			"--policy-document", "file://"+permissionPolicyFile)
		var perrb bytes.Buffer
		permissionsCmd.Stderr = &perrb
		if err := permissionsCmd.Run(); err != nil {
			log.Println(perrb.String())
			return errors.New(perrb.String())
		}
		return nil
	})
}
