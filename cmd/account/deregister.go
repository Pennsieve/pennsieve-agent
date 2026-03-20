package account

import (
	"context"
	"fmt"
	"strings"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var DeregisterCmd = &cobra.Command{
	Use:   "deregister",
	Short: "Deregister an account from Pennsieve",
	Run: func(cmd *cobra.Command, args []string) {
		accountType, _ := cmd.Flags().GetString("type")
		profile, _ := cmd.Flags().GetString("profile")
		force, _ := cmd.Flags().GetBool("force")

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
		resp, err := client.Deregister(context.Background(), &req)
		if err != nil {
			if strings.Contains(err.Error(), "active compute nodes") {
				fmt.Println("Error:", err.Error())
				fmt.Println("\nUse --force to deregister anyway.")
				return
			}
			shared.HandleAgentError(err, fmt.Sprintf("error: Unable to complete Deregister command: %v", err))
			return
		}

		fmt.Printf("Account %s deregistered. IAM role %s deleted.\n", resp.AccountId, resp.RoleName)
	},
}