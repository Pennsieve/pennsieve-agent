package account

import (
	"context"
	"fmt"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var RegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register accounts as compute nodes",
	Long: `Connects the AWS account of the profile to Pennsieve: creates Pennsieve's role
in it from Pennsieve's CloudFormation template (a stack named after the role),
with the profile's credentials, then registers the account. The profile needs
permission to create CloudFormation stacks and IAM roles.

The same template is what the Pennsieve app's "Connect AWS account" uses.`,
	Run: func(cmd *cobra.Command, args []string) {
		accountType, _ := cmd.Flags().GetString("type")
		profile, _ := cmd.Flags().GetString("profile")

		value, ok := api.Account_AccountType_value[accountType]
		if !ok {
			panic("invalid accountType value")
		}

		req := api.RegisterRequest{
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
		fmt.Println("Creating Pennsieve's role in your AWS account from its CloudFormation template; this takes a minute or two…")
		registerResponse, err := client.Register(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("error: Unable to complete Register command: %v", err))
			return
		}

		fmt.Printf("Account %s Registered\n", registerResponse.AccountId)
	},
}
