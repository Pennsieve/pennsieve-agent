package aws

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/pennsieve/pennsieve-agent/internal/account"
	"github.com/pennsieve/pennsieve-agent/internal/projectpath"
)

type AWSRoleManager struct {
	AccountId string
	Profile   string
}

func NewAWSRoleManager(pennsieveAccountId string, profile string) account.Registration {
	return &AWSRoleManager{AccountId: pennsieveAccountId, Profile: profile}
}

func (r *AWSRoleManager) Create() ([]byte, error) {
	roleName := fmt.Sprintf("ROLE-%s", r.AccountId)

	// create trust policy
	trustPolicy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {
						"AWS": "%s"
					},
					"Action": "sts:AssumeRole"
				}
			]
		}`, r.AccountId)
	trustPolicyFile := fmt.Sprintf("TRUST_POLICY_%s.json", r.AccountId)

	err := os.WriteFile(fmt.Sprintf("%s/internal/aws/%s", projectpath.Root, trustPolicyFile),
		[]byte(trustPolicy), 0644)
	if err != nil {
		log.Println("error writing data:", err)
		return nil, err
	}

	// create role
	cmd := exec.Command("aws",
		"--profile", r.Profile,
		"iam", "create-role",
		"--role-name", roleName,
		"--assume-role-policy-document", fmt.Sprintf("file://%s", trustPolicyFile))
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	cmd.Dir = fmt.Sprintf("%s/internal/aws", projectpath.Root)
	err = cmd.Run()
	if err != nil {
		log.Println(errb.String())
		if strings.Contains(errb.String(), "EntityAlreadyExists") {
			return nil, errors.New("role already exists")
		}
		return nil, errors.New(errb.String())
	}

	// attach inline permissions
	permissionPolicyFile := "PERMISSION_POLICY.json"
	policyName := fmt.Sprintf("POLICY-%s", r.AccountId)
	permissionsCmd := exec.Command("aws",
		"--profile", r.Profile,
		"iam", "put-role-policy",
		"--policy-name", policyName,
		"--role-name", roleName,
		"--policy-document", fmt.Sprintf("file://%s", permissionPolicyFile))
	var poutb, perrb bytes.Buffer
	permissionsCmd.Stdout = &poutb
	permissionsCmd.Stderr = &perrb
	permissionsCmd.Dir = fmt.Sprintf("%s/internal/aws", projectpath.Root)
	err = permissionsCmd.Run()
	if err != nil {
		log.Println(perrb.String())
		return nil, errors.New(perrb.String())
	}

	return outb.Bytes(), nil
}
