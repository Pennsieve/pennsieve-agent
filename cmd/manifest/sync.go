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

var SyncCmd = &cobra.Command{
	Use:   "sync [flags] [MANIFEST ID] ",
	Short: "Syncs manifest with server.",
	Long:  `Syncs manifest with server.`,
	Run: func(cmd *cobra.Command, args []string) {

		req := pb.SyncManifestRequest{
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
		manifestResponse, err := client.SyncManifest(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
			return
		}

		fmt.Printf("Synced Manifest: %d added and %d removed",
			manifestResponse.NrFilesAdded, manifestResponse.NrFilesRemoved)
	},
}

func init() {
}
