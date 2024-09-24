package fetch

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
	Use:   "fetch [dataset_id] [target_path]",
	Short: "Map a Pennsieve Dataset to a local folder.",
	Long: `
  The fetch command creates a folder that is associated with a 
  Pennsieve dataset. Subfolders and files will be created within
  the target folder to match the dataset structure on the platfom.
  
  In contrast to the download command, the files will NOT be 
  downloaded to the local machine, but instead, an empty file 
  representing the file on Pennsieve will be created. Use the "sync" 
  command to download specific files, or folders to your local
  machine.

  Using "fetch" and "sync" allows for users to efficiently map 
  Pennsieve datasets to their local machines without requiring
  to download each file in the dataset. This saves space, and costs
  associated with downloading the entire dataset.
  `,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		datasetId := args[0]

		folder := args[1]

		// Check and make path absolute
		absPath, err := shared.GetAbsolutePath(folder)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
			return
		}

		fetchRequest := api.FetchRequest{
			DatasetId:    datasetId,
			TargetFolder: absPath,
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
			fmt.Println("Requested Fetch of dataset: ", datasetId)
		} else {
			fmt.Println("Unable to request download command: ", fetchResponse.Status)
			log.Errorf("Unable to request download command: %v", fetchResponse.Status)
		}
	},
}

func init() {

}
