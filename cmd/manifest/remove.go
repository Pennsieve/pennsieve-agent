package manifest

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

var RemoveCmd = &cobra.Command{
	Use:   "remove <MANIFEST-ID> <ID> [...ID]",
	Short: "Removes files from an existing manifest.",
	Long:  `Creates manifest for upload.`,
	Run: func(cmd *cobra.Command, args []string) {

		manifestId, _ := cmd.Flags().GetInt32("manifest_id")
		fmt.Println("manifest if ", manifestId)
		if manifestId == -1 {
			log.Fatalln("Need to specify manifest id with `manifest_id` flag.")
		}

		fmt.Println(args[0])

		req := pb.RemoveFromManifestRequest{
			ManifestId: manifestId,
			RemovePath: args[0],
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		manifestResponse, err := client.RemoveFromManifest(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Remove Manifest command: %v", err))
			return
		}

		fmt.Println(manifestResponse.Status)
	},
}

func init() {
	RemoveCmd.Flags().Int32P("manifest_id", "m",
		0, "Manifest ID.")

	RemoveCmd.MarkFlagRequired("manifest_id")

}
