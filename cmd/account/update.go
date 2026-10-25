package account

import (
	"context"
	"fmt"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/cmd/shared"
	"github.com/pennsieve/pennsieve-agent/v2/internal/account"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var UpdateRoleCmd = &cobra.Command{
	Use:   "update-role",
	Short: "Update a registered account's role to Pennsieve's latest CloudFormation template",
	Long: `Updates the account's role to Pennsieve's latest CloudFormation template, with
the AWS profile's credentials.

An account connected with CloudFormation has its stack updated in place. An
older account (its role made before the template) gets a new role from the
template, and Pennsieve switches the account to it; delete the old role
afterwards (the command names it).`,
	Run: func(cmd *cobra.Command, args []string) {
		accountType, _ := cmd.Flags().GetString("type")
		profile, _ := cmd.Flags().GetString("profile")

		value, ok := api.Account_AccountType_value[accountType]
		if !ok {
			fmt.Println("Error: invalid account type:", accountType)
			return
		}

		req := api.UpdateRoleRequest{
			Account:     &api.Account{Type: api.Account_AccountType(value)},
			Credentials: &api.Credentials{Profile: profile},
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial("127.0.0.1:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		fmt.Println("Updating the role's CloudFormation stack in your AWS account; this takes a minute or two…")
		var trailer metadata.MD
		resp, err := client.UpdateRole(context.Background(), &req, grpc.Trailer(&trailer))
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("error: Unable to complete UpdateRole command: %v", err))
			return
		}

		if old := trailer.Get(account.TrailerOldRoleName); len(old) > 0 && old[0] != "" {
			fmt.Printf("Account %s now uses role %s, from Pennsieve's CloudFormation template.\n", resp.AccountId, resp.RoleName)
			fmt.Printf("Delete the old role %s from your AWS account (IAM console, or: aws --profile %s iam delete-role-policy --role-name %s --policy-name POLICY-%s && aws --profile %s iam delete-role --role-name %s).\n",
				old[0], profile, old[0], old[0], profile, old[0])
			fmt.Println("Pennsieve can't delete it for you: the role's own policy forbids Pennsieve to touch Pennsieve-Compute roles.")
			return
		}
		fmt.Printf("Role %s for account %s updated\n", resp.RoleName, resp.AccountId)
	},
}
