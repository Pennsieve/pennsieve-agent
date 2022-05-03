package manifest

import (
	"context"
	"fmt"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ManifestCmd = &cobra.Command{
	Use:   "manifest [flags] [PATH] [...PATH]",
	Short: "Creates manifest for upload.",
	Long:  `Creates manifest for upload.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Manifest Create called")

		req := pb.CreateManifestRequest{
			BasePath:  args[0],
			Recursive: true,
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		manifestResponse, err := client.CreateUploadManifest(context.Background(), &req)
		if err != nil {
			fmt.Println("Error creating manifest: ", err)
		}
		fmt.Println(manifestResponse)

	},
}

func init() {
	ManifestCmd.AddCommand(ListCmd)

}
