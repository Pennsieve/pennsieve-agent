package download

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/cmd/shared"
	pkgshared "github.com/pennsieve/pennsieve-agent/v2/pkg/shared"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var DatasetCmd = &cobra.Command{
	Use:   "dataset [dataset-id] [target-folder]",
	Short: "Download dataset.",
	Long: `Download dataset to the selected folder. A new dataset folder will be created in the selected target folder.

Use --node to download only some folders or packages of the dataset, for example:

  pennsieve download dataset N:dataset:1234 ./data --node N:collection:5678 --node N:package:9012`,
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

		nodeIds, _ := cmd.Flags().GetStringSlice("node")
		force, _ := cmd.Flags().GetBool("force")

		req := api.DownloadDatasetRequest{
			DatasetId:    datasetId,
			TargetFolder: absPath,
			NodeIds:      nodeIds,
			Force:        force,
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
		if downloadResponse.Status == "Success" {
			fmt.Printf("Downloading %d files (%s) of %s to %s\n",
				downloadResponse.FileCount, pkgshared.HumanBytes(downloadResponse.TotalBytes), datasetId, absPath)
			fmt.Println("Follow progress with: pennsieve agent subscribe")
			fmt.Println("Cancel with: pennsieve download cancel " + datasetId)
		} else {
			fmt.Println("Unable to request download command: ", downloadResponse.Status)
			log.Errorf("Unable to request download command: %v", downloadResponse.Status)
		}
	},
}

func init() {
	DatasetCmd.Flags().StringSlice("node", nil,
		"Download only these folders or packages (node ids; repeat or separate with commas)")
	DatasetCmd.Flags().Bool("force", false,
		"Download even when the target folder's disk has too little free space")
}
