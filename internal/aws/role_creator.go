package aws

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/pennsieve/pennsieve-agent/internal/account"
	"github.com/pennsieve/pennsieve-agent/internal/projectpath"
)

type AWSRoleCreator struct {
	AccountId string
	Profile   string
}

func NewAWSRoleCreator(accountId string, profile string) account.Registration {
	return &AWSRoleCreator{AccountId: accountId, Profile: profile}
}

func (r *AWSRoleCreator) Create() ([]byte, error) {
	// create role
	cmd := exec.Command("./create-role.sh", fmt.Sprintf("%v", r.AccountId), r.Profile)
	cmd.Dir = fmt.Sprintf("%s/internal/aws/scripts", projectpath.Root)
	out, err := cmd.Output()
	if err != nil {
		log.Println(string(out))
		return nil, err
	}
	log.Println(string(out))

	if strings.TrimSpace(string(out)) == "ROLE EXISTS" {
		return nil, errors.New("role already exists")
	}

	data, err := os.ReadFile(fmt.Sprintf("%s/internal/aws/scripts/role-%v.json",
		projectpath.Root, r.AccountId))
	if err != nil {
		return nil, err
	}

	return data, nil
}
