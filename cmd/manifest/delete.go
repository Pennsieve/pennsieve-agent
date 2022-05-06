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

var DeleteCmd = &cobra.Command{
	Use:   "delete <manifest_id>",
	Short: "Deletes existing manifest.",
	Long:  `Deletes existing manifest.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req := pb.DeleteManifestRequest{
			ManifestId: args[0],
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		manifestResponse, err := client.DeleteUploadManifest(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
			return
		}

		fmt.Println(manifestResponse)
	},
}

func init() {

}
