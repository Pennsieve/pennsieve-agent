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
	Short: "Connect an AWS account to Pennsieve from the command line (the app is the recommended way)",
	Long: `The recommended way to connect an AWS account is in the Pennsieve app:
Analysis → Compute Nodes → Connect AWS account. This command does the same
from the command line, with your local AWS profile, for scripts and automation.

It creates Pennsieve's role in the profile's AWS account from Pennsieve's
CloudFormation template (a stack named after the role), with the profile's
credentials, then registers the account. The profile needs permission to
create CloudFormation stacks and IAM roles.`,
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
		fmt.Println("The account is connected; manage it in the Pennsieve app (Analysis → Compute Nodes).")
	},
}
