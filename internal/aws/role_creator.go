package aws

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/pennsieve/pennsieve-agent/internal/account"
)

type AWSRoleCreator struct {
	AccountId int64
	Profile   string
}

func NewAWSRoleCreator(accountId int64, profile string) account.Registration {
	return &AWSRoleCreator{AccountId: accountId, Profile: profile}
}

func (r *AWSRoleCreator) Create() ([]byte, error) {
	// create role
	cmd := exec.Command("./scripts/aws/create-role.sh", fmt.Sprintf("%v", r.AccountId), r.Profile)
	cmd.Dir = "./pkg/server"
	out, err := cmd.Output()
	if err != nil {
		log.Println(string(out))
		return nil, err
	}
	fmt.Println(string(out))

	data, err := os.ReadFile(fmt.Sprintf("./pkg/server/scripts/aws/role-%v.json", r.AccountId))
	if err != nil {
		return nil, err
	}

	return data, nil
}
