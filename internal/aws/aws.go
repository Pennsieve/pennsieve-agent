package aws

import (
	"fmt"
	"log"
	"os/exec"
)

func CreateRole(accountId int64, profile string) {
	cmd := exec.Command("./scripts/aws/create-role.sh", fmt.Sprintf("%v", accountId), profile)
	cmd.Dir = "./pkg/server"
	out, err := cmd.Output()
	if err != nil {
		log.Println(string(out))
		log.Println(err)
	}
	fmt.Println(string(out))
}
