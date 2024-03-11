package account

import (
	"context"
	"fmt"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var RegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register accounts as compute nodes",
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

		registerResponse, err := client.Register(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("error: Unable to complete Register command: %v", err))
			return
		}

		fmt.Println("Account Registration")
		fmt.Println(registerResponse)

	},
}
