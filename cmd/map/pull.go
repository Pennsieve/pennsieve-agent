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

var PullCmd = &cobra.Command{
	Use:   "pull [target_path]",
	Short: "Pull files from the server into a mapped Pennsieve Dataset.",
	Long: `
  [BETA] This feature is in Beta mode and is currently still undergoing
  testing and optimization.

  The "pull" command downloads files from the Pennsieve platform 
  in Pennsieve managed folders on your local machine. If you have
  mapped a Pennsieve dataset to a local folder using the "fetch" 
  command, you can use "pull" to download files individually or
  per folder in the mapped dataset.
  `,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target_path := args[0]

		// Check and make path absolute
		absPath, err := shared.GetAbsolutePath(target_path)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
			return
		}

		pullRequest := api.PullRequest{
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
		pullResponse, err := client.Pull(context.Background(), &pullRequest)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Fetch command: %v", err))
			return
		}
		if pullResponse.Status == "Success" {
			fmt.Println("success")
		} else {
			fmt.Println("Unable to request pull command: ", pullResponse.Status)
			log.Errorf("Unable to request pull command: %v", pullResponse.Status)
		}

	},
}

func init() {

}
