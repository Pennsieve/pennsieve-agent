package manifest

import (
	"context"
	"fmt"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var CreateCmd = &cobra.Command{
	Use:   "create [flags] [PATH] [...PATH]",
	Short: "Creates manifest for upload.",
	Long:  `Creates manifest for upload.`,
	Run: func(cmd *cobra.Command, args []string) {

		targetBasePath, _ := cmd.Flags().GetString("target_path")

		req := pb.CreateManifestRequest{
			BasePath:       args[0],
			TargetBasePath: targetBasePath,
			Recursive:      true,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		manifestResponse, err := client.CreateManifest(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
			return
		}

		fmt.Println("Manifest ID:", manifestResponse.ManifestId, "Message:", manifestResponse.Message)
	},
}

func init() {
	CreateCmd.Flags().StringP("target_path", "t",
		"", "Target base path in dataset.")
}
