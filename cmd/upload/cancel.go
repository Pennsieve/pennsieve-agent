package upload

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strconv"
)

var CancelCmd = &cobra.Command{
	Use:   "cancel <manifestId>",
	Short: "Cancel upload session.",
	Long:  `Cancel upload session.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		i, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			fmt.Printf("Error: <manifestId> should be an integer.")
			return
		}
		selectedManifest := int32(i)

		// If no manifest is specified, cancel all running upload sessions.
		cancelAll := false

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
			shared.HandleAgentError(err, fmt.Sprintf("Error uploading file: %v", err))
		}
		fmt.Println(uploadResponse)

	},
}

func init() {

}
