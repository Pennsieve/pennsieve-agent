package aws

import (
	"fmt"
	"log"
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

func (r *AWSRoleCreator) Create() error {
	cmd := exec.Command("./scripts/aws/create-role.sh", fmt.Sprintf("%v", r.AccountId), r.Profile)
	cmd.Dir = "./pkg/server"
	out, err := cmd.Output()
	if err != nil {
		log.Println(string(out))
		return err
	}

	fmt.Println(string(out))
	return nil
}
