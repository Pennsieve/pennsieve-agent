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
	AccountId string
	Profile   string
}

func NewAWSRoleManager(pennsieveAccountId string, profile string) account.Registration {
	return &AWSRoleManager{AccountId: pennsieveAccountId, Profile: profile}
}

func (r *AWSRoleManager) Create() ([]byte, error) {
	roleName := fmt.Sprintf("ROLE-%s", r.AccountId)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("error getting home directory:", err)
		return nil, err
	}
	pennsieveFolder := filepath.Join(home, ".pennsieve")

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

	trustPolicyFileLocation := fmt.Sprintf("%s/%s", pennsieveFolder, trustPolicyFile)
	err = os.WriteFile(trustPolicyFileLocation, []byte(trustPolicy), 0644)
	if err != nil {
		log.Println("error writing trust policy data:", err)
		return nil, err
	}

	// create permission policy
	permissionPolicy := `{
		"Version": "2012-10-17",
    	"Statement": [
        {
            "Effect": "Allow",
            "Action": "*",
            "Resource": "*"
        }
    ]
	}`
	permissionPolicyFile := "PERMISSION_POLICY.json"

	permissionPolicyFileLocation := fmt.Sprintf("%s/%s", pennsieveFolder, permissionPolicyFile)
	err = os.WriteFile(permissionPolicyFileLocation, []byte(permissionPolicy), 0644)
	if err != nil {
		log.Println("error writing permission policy data:", err)
		return nil, err
	}

	// create role
	cmd := exec.Command("aws",
		"--profile", r.Profile,
		"iam", "create-role",
		"--role-name", roleName,
		"--assume-role-policy-document", fmt.Sprintf("file://%s", trustPolicyFileLocation))
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err = cmd.Run()
	if err != nil {
		log.Println(errb.String())
		if strings.Contains(errb.String(), "EntityAlreadyExists") {
			return nil, errors.New("role already exists")
		}
		return nil, errors.New(errb.String())
	}

	// attach inline permissions
	policyName := fmt.Sprintf("POLICY-%s", r.AccountId)
	permissionsCmd := exec.Command("aws",
		"--profile", r.Profile,
		"iam", "put-role-policy",
		"--policy-name", policyName,
		"--role-name", roleName,
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
