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
	"strconv"
)

var SyncCmd = &cobra.Command{
	Use:   "sync [flags] [MANIFEST ID] ",
	Short: "Syncs manifest with server.",
	Long:  `Syncs manifest with server.`,
	Run: func(cmd *cobra.Command, args []string) {

		i, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			panic(err)
		}
		manifestId := int32(i)

		req := pb.SyncManifestRequest{
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
		manifestResponse, err := client.SyncManifest(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
			return
		}

		fmt.Printf("Synced Manifest: %s -- %d updated, %d removed, and %d failed.",
			manifestResponse.ManifestNodeId, manifestResponse.NrFilesUpdated, manifestResponse.NrFilesRemoved, manifestResponse.NrFilesFailed)
	},
}

func init() {
}
