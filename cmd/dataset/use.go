package dataset

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/config"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"log"
)

var UseCmd = &cobra.Command{
	Use:   "use <dataset>",
	Short: "Set your current working dataset.",
	Long:  `Set your current working dataset.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		datasetId := args[0]

		req := v1.UseDatasetRequest{
			DatasetId: datasetId,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		ctx := context.Background()

		// Update active dataset using GRPC
		client := v1.NewAgentClient(conn)
		useDatasetResponse, err := client.UseDataset(ctx, &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
			return
		}

		// Get the dataset directly from service to render
		db, _ := config.InitializeDB()
		userSettingsStore := store.NewUserSettingsStore(db)
		userInfoStore := store.NewUserInfoStore(db)
		pennsieveClient, err := config.InitPennsieveClient(userSettingsStore, userInfoStore)
		if err != nil {
			log.Fatalln("Cannot connect to Pennsieve.")
		}
		response, err := pennsieveClient.Dataset.Get(ctx, useDatasetResponse.DatasetId)
		if err != nil {
			fmt.Println("Error fetching dataset from Pennsieve: ", useDatasetResponse.DatasetId)
			log.Println("CMD:Dataset:Use: ", err)
			return
		}

		PrettyPrint(response, false)
	},
}
