package upload

import (
	"context"
	"fmt"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

var CancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel upload session.",
	Long:  `Cancel upload session.`,
	Run: func(cmd *cobra.Command, args []string) {

		selectedManifest, err := cmd.Flags().GetString("manifest")
		if err != nil {
			log.Fatalln("Error getting manifest flag from command line: ", err)
		}

		// If no manifest is specified, cancel all running upload sessions.
		cancelAll := false
		if selectedManifest == "" {
			fmt.Println("Cancelling all upload sessions.")
			cancelAll = true
		} else {
			fmt.Println("Cancelling upload session: ", selectedManifest)
		}

		req := pb.CancelUploadRequest{
			ManifestId: selectedManifest,
			CancelAll:  cancelAll,
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		uploadResponse, err := client.CancelUpload(context.Background(), &req)
		if err != nil {
			fmt.Println("Error uploading file: ", err)
		}
		fmt.Println(uploadResponse)

	},
}

func init() {
	CancelCmd.Flags().StringP("manifest", "m", "",
		"Specify manifest id to be cancelled")
}
