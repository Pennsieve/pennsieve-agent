package download

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

var DatasetCmd = &cobra.Command{
	Use:   "dataset [dataset-id] [target-folder]",
	Short: "Download dataset.",
	Long:  `Download dataset to the selected folder. A new dataset folder will be created in the selected target folder.`,
	Args:  cobra.MinimumNArgs(2),
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

		req := api.DownloadDatasetRequest{
			DatasetId:    datasetId,
			TargetFolder: absPath,
		}

		downloadReq := api.DownloadRequest{
			Type: api.DownloadRequest_DATASET,
			Data: &api.DownloadRequest_Dataset{Dataset: &req},
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		downloadResponse, err := client.Download(context.Background(), &downloadReq)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Download command: %v", err))
			return
		}
		fmt.Println(downloadResponse)
		if downloadResponse.Status == "Success" {
			fmt.Println("Requested Download of dataset: ", datasetId)
		} else {
			fmt.Println("Unable to request download command: ", downloadResponse.Status)
			log.Errorf("Unable to request download command: %v", downloadResponse.Status)
		}
	},
}

func init() {
}
