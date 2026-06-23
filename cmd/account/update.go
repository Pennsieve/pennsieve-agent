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

var UpdateRoleCmd = &cobra.Command{
	Use:   "update-role",
	Short: "Re-sync a registered account's IAM role policy with the latest from Pennsieve",
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
		resp, err := client.UpdateRole(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("error: Unable to complete UpdateRole command: %v", err))
			return
		}

		fmt.Printf("Role %s for account %s updated\n", resp.RoleName, resp.AccountId)
	},
}
