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
	"log"
)

var ResetCmd = &cobra.Command{
	Use:   "reset <MANIFEST-ID> <ID> [...ID]",
	Short: "Resets status of all files in a manifest",
	Long:  `Resets status of all files in a manifest.`,
	Run: func(cmd *cobra.Command, args []string) {

		manifestId, _ := cmd.Flags().GetInt32("manifest_id")
		if manifestId == -1 {
			log.Fatalln("Need to specify manifest id with `manifest_id` flag.")
		}

		req := pb.ResetManifestRequest{
			ManifestId: manifestId,
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		manifestResponse, err := client.ResetManifest(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
			return
		}

		fmt.Println(manifestResponse.Status)
	},
}

func init() {
	ResetCmd.Flags().Int32P("manifest_id", "m",
		0, "Manifest ID.")

	ResetCmd.MarkFlagRequired("manifest_id")

}
