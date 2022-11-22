package manifest

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/pennsieve/pennsieve-agent/pkg/subscriber"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"math/rand"
	"strconv"
	"time"
)

var SyncCmd = &cobra.Command{
	Use:   "sync [flags] [MANIFEST ID] ",
	Short: "Syncs manifest with server.",
	Long:  `Synchronizes the manifest with the Pennsieve platform. `,
	Run: func(cmd *cobra.Command, args []string) {

		i, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			panic(err)
		}
		manifestId := int32(i)

		req := api.SyncManifestRequest{
			ManifestId: manifestId,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		_, err = client.SyncManifest(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Sync Manifest command: %v", err))
			return
		}

		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		SubscribeClient, err := subscriber.NewSubscriberClient(int32(r1.Intn(100)))
		if err != nil {
			log.Fatal(err)
		}
		// Dispatch client goroutine
		fmt.Printf("Synchronizing manifest.\n You can safely Ctr-C as synchronization will continue to run in the background.")
		fmt.Println("\n\nUse " +
			"\"pennsieve agent subscribe\" to track all events from the Pennsieve Agent.")

		fmt.Println("\n------------")
		SubscribeClient.Start([]api.SubscribeResponse_MessageType{api.SubscribeResponse_SYNC_STATUS}, subscriber.StopOnStatus{
			Enable: true,
			OnType: []api.SubscribeResponse_MessageType{api.SubscribeResponse_SYNC_STATUS},
		})

	},
}

func init() {
}
