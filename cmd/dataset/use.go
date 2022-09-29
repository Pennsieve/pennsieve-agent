package dataset

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/cmd/config"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"log"
)

// whoamiCmd represents the whoami command
var UseCmd = &cobra.Command{
	Use:   "use <dataset>",
	Short: "Set your current working dataset.",
	Long:  `Set your current working dataset.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.InitDB()
	},
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		datasetId := args[0]

		req := pb.UseDatasetRequest{
			DatasetId: datasetId,
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		// Update active dataset using GRPC
		client := pb.NewAgentClient(conn)
		useDatasetResponse, err := client.UseDataset(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
			return
		}

		// Get the dataset directly from service to render
		pennsieveClient := api.PennsieveClient
		response, err := pennsieveClient.Dataset.Get(nil, useDatasetResponse.DatasetId)
		if err != nil {
			log.Println(err)
			log.Fatalln("Unknown dataset: ", useDatasetResponse.DatasetId)
		}

		PrettyPrint(response, false)
	},
}
