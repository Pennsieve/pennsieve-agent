package _map

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

var PushCmd = &cobra.Command{
	Use:   "push [target_path]",
	Short: "Push local changes to the remote Pennsieve Dataset",
	Long: `
  [BETA] This feature is in Beta mode and is currently still undergoing
  testing and optimization.

  Push identifies new files in your local mapped dataset and uploads them
  to Pennsieve while preserving the directory structure.
  `,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Determine the target folder
		var folder string
		if len(args) > 0 {
			folder = args[0]
		} else {
			folder = "."
		}

		// Check and make path absolute
		absPath, err := shared.GetAbsolutePath(folder)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
			return
		}

		// Create a push request
		pushRequest := api.PushRequest{
			Path: absPath,
		}

		// Connect to the agent server
		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		pushResponse, err := client.Push(context.Background(), &pushRequest)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Push command: %v", err))
			return
		}

		fmt.Println(pushResponse.Status)
	},
}

func init() {

}
