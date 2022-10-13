package manifest

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

		req := v1.ResetManifestRequest{
			ManifestId: manifestId,
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := v1.NewAgentClient(conn)
		manifestResponse, err := client.ResetManifest(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Reset Manifest command: %v", err))
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
