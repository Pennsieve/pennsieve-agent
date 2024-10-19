package _map

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var FetchCmd = &cobra.Command{
	Use:   "fetch [target_path]",
	Short: "Fetch remote state to locally mapped dataset",
	Long: `
  [BETA] This feature is in Beta mode and is currently still undergoing
  testing and optimization.

  The 'fetch' command will fetch remote state to locally mapped dataset. If there are
  changes in the dataset on the server, they will be mapped to the local state.

  `,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		folder := args[0]

		// Check and make path absolute
		absPath, err := shared.GetAbsolutePath(folder)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
			return
		}

		fetchRequest := api.FetchRequest{
			Path: absPath,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		fetchResponse, err := client.Fetch(context.Background(), &fetchRequest)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Fetch command: %v", err))
			return
		}
		if fetchResponse.Status == "Success" {
			fmt.Println("Requested Fetch of dataset. ")
		} else {
			fmt.Println("Unable to request download command: ", fetchResponse.Status)
			log.Errorf("Unable to request download command: %v", fetchResponse.Status)
		}
	},
}

func init() {

}
