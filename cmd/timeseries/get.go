package timeseries

import (
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/config"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"strconv"
	"time"
)

var getCmd = &cobra.Command{
	Use:   "get <PACKAGE-ID> <START-TIMESTAMP> <END-TIMESTAMP> <TARGET>",
	Short: "Retrieve timeseries data as CSV file.",
	Long: `Retrieve timeseries data as CSV file. 
The command will return a CSV file at the TARGET location with the requested range of timeseries data.

`,
	Args: cobra.MinimumNArgs(4),
	Run: func(cmd *cobra.Command, args []string) {

		packageId := args[0]
		startTimestamp, _ := strconv.ParseUint(args[1], 10, 64)
		endTimestamp, _ := strconv.ParseUint(args[2], 10, 64)
		//target := args[3]

		db, _ := config.InitializeDB()
		userSettingsStore := store.NewUserSettingsStore(db)
		s, _ := userSettingsStore.Get()
		if len(s.UseDatasetId) == 0 {
			fmt.Println("\nError: No dataset specified; use 'pennsieve dataset use <node-id>' to set active dataset.")
			return
		}

		log.Info("datasetID")

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock()}...)

		client := api.NewAgentClient(conn)

		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		stream, err := client.GetTimeseriesRangeForChannels(ctx, &api.GetTimeseriesRangeRequest{
			DatasetId: s.UseDatasetId,
			PackageId: packageId,
			ChannelId: nil,
			StartTime: startTimestamp,
			EndTime:   endTimestamp,
			Refresh:   false,
		})
		if err != nil {
			log.Error("Error getting timeseries range: ", err)
			return
		}

		for {
			req, err := stream.Recv()
			if err == io.EOF {
				fmt.Println("Closing")
				// End of stream, process accumulated data and send response

				stream.CloseSend()
				break
			}
			// Process the received request message
			fmt.Println(req)
		}

	},
}

func init() {
}
