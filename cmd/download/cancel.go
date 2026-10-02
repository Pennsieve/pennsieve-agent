package download

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var CancelCmd = &cobra.Command{
	Use:   "cancel [dataset-or-package-id]",
	Short: "Cancel a download.",
	Long: `Cancel the running download of a dataset (including map pull) or package, or
all downloads with --all. Files being downloaded stop too; files already
downloaded are kept.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cancelAll, _ := cmd.Flags().GetBool("all")
		if len(args) == 0 && !cancelAll {
			fmt.Println("Name the dataset or package whose download to cancel, or use --all.")
			return
		}

		req := api.CancelDownloadRequest{CancelAll: cancelAll}
		if len(args) > 0 {
			req.Id = &args[0]
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		resp, err := client.CancelDownload(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error cancelling the download: %v", err))
			return
		}
		fmt.Println(resp.Status)
	},
}

func init() {
	CancelCmd.Flags().Bool("all", false, "Cancel all running downloads")
}
