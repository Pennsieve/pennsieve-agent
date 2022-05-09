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

var RemoveCmd = &cobra.Command{
	Use:   "remove <MANIFEST-ID> <ID> [...ID]",
	Short: "Removes files from an existing manifest.",
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

		fmt.Println(manifestResponse.Status)
	},
}

func init() {

}
