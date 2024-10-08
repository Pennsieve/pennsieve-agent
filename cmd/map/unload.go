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

var UnloadCmd = &cobra.Command{
	Use:   "unload [target_path]",
	Short: "Remove local copy of mapped files.",
	Long: `
  [BETA] This feature is in Beta mode and is currently still undergoing
  testing and optimization.

  The "unload" command removes downloaded files from the local machine
  and reverts the files to 'empty' placeholders on the machine. The path 
  parameter can either be a file, or a folder.
  `,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetPath := args[0]

		// Check and make path absolute
		absPath, err := shared.GetAbsolutePath(targetPath)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
			return
		}

		revertRequest := api.UnloadRequest{
			Path: absPath,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.NewClient(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		pullResponse, err := client.Unload(context.Background(), &revertRequest)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Revert command: %v", err))
			return
		}
		if pullResponse.Status == "Success" {
			fmt.Println("success")
		} else {
			fmt.Println("Unable to request revert command: ", pullResponse.Status)
			log.Errorf("Unable to request revert command: %v", pullResponse.Status)
		}

	},
}

func init() {

}
