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

var PackageCmd = &cobra.Command{
	Use:   "package [package-id]",
	Short: "(Default) Download package.",
	Long:  `Download package.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		packageId := args[0]

		getPresignedUrl, _ := cmd.Flags().GetBool("presigned")

		req := api.DownloadPackageRequest{
			PackageId:       packageId,
			GetPresignedUrl: getPresignedUrl,
		}

		downloadReq := api.DownloadRequest{
			Type: api.DownloadRequest_PACKAGE,
			Data: &api.DownloadRequest_Package{Package: &req},
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
			fmt.Println("Requested Download of package: ", packageId)
		} else {
			fmt.Println("Unable to request download command: ", downloadResponse.Status)
			log.Errorf("Unable to request download command: %v", downloadResponse.Status)
		}
	},
}

func init() {
}
