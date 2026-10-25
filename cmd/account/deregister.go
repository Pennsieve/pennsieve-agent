package account

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/cmd/shared"
	"github.com/pennsieve/pennsieve-agent/v2/internal/account"
	"github.com/pennsieve/pennsieve-agent/v2/internal/aws"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var DeregisterCmd = &cobra.Command{
	Use:   "deregister",
	Short: "Deregister an account from Pennsieve",
	Run: func(cmd *cobra.Command, args []string) {
		accountType, _ := cmd.Flags().GetString("type")
		profile, _ := cmd.Flags().GetString("profile")
		force, _ := cmd.Flags().GetBool("force")
		deleteStack, _ := cmd.Flags().GetBool("delete-stack")

		value, ok := api.Account_AccountType_value[accountType]
		if !ok {
			fmt.Println("Error: invalid account type:", accountType)
			return
		}

		req := api.DeregisterRequest{
			Account:     &api.Account{Type: api.Account_AccountType(value)},
			Credentials: &api.Credentials{Profile: profile},
			Force:       force,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial("127.0.0.1:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		var trailer metadata.MD
		resp, err := client.Deregister(context.Background(), &req, grpc.Trailer(&trailer))
		if err != nil {
			if strings.Contains(err.Error(), "active compute nodes") {
				fmt.Println("Error:", err.Error())
				fmt.Println("\nUse --force to deregister anyway.")
				return
			}
			shared.HandleAgentError(err, fmt.Sprintf("error: Unable to complete Deregister command: %v", err))
			return
		}

		stack := trailer.Get(account.TrailerRoleStack)
		if len(stack) == 0 || stack[0] == "" {
			fmt.Printf("Account %s deregistered. IAM role %s deleted.\n", resp.AccountId, resp.RoleName)
			return
		}

		// The role is a CloudFormation stack: deleting the stack deletes it.
		fmt.Printf("Account %s deregistered.\n", resp.AccountId)
		if !deleteStack && !confirm(fmt.Sprintf("Delete the CloudFormation stack %s, and the role %s with it? [y/N] ", stack[0], resp.RoleName)) {
			fmt.Printf("Kept the stack %s. Delete it in the AWS console (CloudFormation) to remove the role.\n", stack[0])
			return
		}
		stacks := aws.NewStackManager(profile)
		stacks.Progress = func(status string) { fmt.Println("  " + status) }
		if err := stacks.Delete(stack[0]); err == nil {
			err = stacks.Wait(stack[0], "DELETE_COMPLETE")
		}
		if err != nil {
			fmt.Println("Error:", err)
			fmt.Printf("Delete the stack %s in the AWS console (CloudFormation) to remove the role.\n", stack[0])
			return
		}
		fmt.Printf("Stack %s and role %s deleted.\n", stack[0], resp.RoleName)
	},
}

// confirm asks a yes/no question on the terminal; anything but y or yes is no.
func confirm(question string) bool {
	fmt.Print(question)
	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}
