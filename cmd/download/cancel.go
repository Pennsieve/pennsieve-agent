package download

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var CancelCmd = &cobra.Command{
	Use:   "cancel <packageId>",
	Short: "Cancel download session.",
	Long:  `Cancel download session.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		selectedPackage := args[0]

		// If no manifest is specified, cancel all running download sessions.
		cancelAll := false

		req := api.CancelDownloadRequest{
			Id:        &selectedPackage,
			CancelAll: cancelAll,
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		uploadResponse, err := client.CancelDownload(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error canceling download file: %v", err))
		}
		fmt.Println(uploadResponse)

	},
}

func init() {

}
