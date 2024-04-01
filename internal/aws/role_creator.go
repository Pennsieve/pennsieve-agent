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
)

type AWSRoleCreator struct {
	AccountId string
	Profile   string
}

func NewAWSRoleCreator(pennsieveAccountId string, profile string) account.Registration {
	return &AWSRoleCreator{AccountId: pennsieveAccountId, Profile: profile}
}

func (r *AWSRoleCreator) Create() ([]byte, error) {
	roleName := fmt.Sprintf("ROLE-%s", r.AccountId)

	cmd := exec.Command("aws", "--profile", r.Profile, "iam", "get-role", "--role-name", roleName)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()

	// check if role exists
	if err == nil && strings.Contains(outb.String(), roleName) {
		log.Println("role exists")
		return nil, errors.New("role exists")
	}

	log.Println(errb.String())
	// check whether role does not exist
	if strings.Contains(errb.String(), "cannot be found") {
		log.Println("role does not exist")

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

		err = os.WriteFile(trustPolicyFile, []byte(trustPolicy), 0644)
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
		err = cmd.Run()
		if err != nil {
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
		err = permissionsCmd.Run()
		if err != nil {
			log.Println(perrb.String())
			return nil, errors.New(perrb.String())
		}

		return outb.Bytes(), nil
	}

	// other errors - exit with error
	return nil, errors.New(errb.String())
}
