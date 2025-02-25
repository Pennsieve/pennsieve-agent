package timeseries

import (
	"encoding/csv"
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
	"os"
	"strconv"
	"time"
)

var getCmd = &cobra.Command{
	Use:   "get <package-id> <channel-id> <start-time> <end-time> [<target>]]",
	Short: "Writes a CSV file from a range of a timeseries package.",
	Long: `This function returns a CSV file with the data for a single channel of a timeseries package. 
The CSV file will have two columns, the first with the timestamps, the second with the values. 
It will have a single header row with the scientific units of the columns.

Users can provide an optional target file-name. If no target is specified, the file name will 
include the channel name and the time-range of the request. 

By default the specified start- and endtimes are defined in uUTC time. By setting the [-r | --relative_time] 
flag, the start- and stoptime are interpreted as relative to the start time of the channel. 

`,
	Args: cobra.MinimumNArgs(4),
	Run: func(cmd *cobra.Command, args []string) {

		packageId := args[0]
		channelId := args[1]
		startTimestamp, _ := strconv.ParseFloat(args[2], 32)
		endTimestamp, _ := strconv.ParseFloat(args[3], 32)

		target := fmt.Sprintf("%s_%s_%s.csv", channelId[len(channelId)-8:],
			args[2], args[3])

		if len(args) > 4 {
			target = args[4]
		}

		doRelativeTime, err := cmd.Flags().GetBool("relative_time")
		if err != nil {
			log.Printf("Error: Cannot get flag \"relative_time\": %s", err)
			return
		}

		db, _ := config.InitializeDB()
		userSettingsStore := store.NewUserSettingsStore(db)
		s, _ := userSettingsStore.Get()
		if len(s.UseDatasetId) == 0 {
			fmt.Println("\nError: No dataset specified; use 'pennsieve dataset use <node-id>' to set active dataset.")
			return
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.NewClient(":"+port, []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}...)

		client := api.NewAgentClient(conn)

		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		stream, err := client.GetTimeseriesRangeForChannels(ctx, &api.GetTimeseriesRangeRequest{
			DatasetId:    s.UseDatasetId,
			PackageId:    packageId,
			ChannelId:    channelId,
			StartTime:    float32(startTimestamp),
			EndTime:      float32(endTimestamp),
			Refresh:      false,
			RelativeTime: doRelativeTime,
		})
		if err != nil {
			log.Error("Error getting timeseries range: ", err)
			return
		}

		file, err := os.Create(target)
		if err != nil {
			log.Fatal("Cannot create file", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		firstBlock := uint64(0)
		for {
			req, err := stream.Recv()

			// Server closes connection when done streaming.
			if err == io.EOF {
				stream.CloseSend()
				break
			}

			// Check Message Type
			switch req.Type {
			case api.GetTimeseriesRangeResponse_CHANNEL_INFO:

				log.Info("Got Channel Info")

				ch := req.GetChannel()
				// Check if this message arrives before first block
				if firstBlock == 0 {
					header := []string{
						ch.Name,
						fmt.Sprintf("%sHz - %s", strconv.FormatFloat(float64(ch.GetRate()), 'f', -1, 64), ch.GetUnit()),
					}
					writer.Write(header)

				} else {
					log.Error("Received channel-info after blocks, ignoring info.")
				}

				break
			case api.GetTimeseriesRangeResponse_RANGE_DATA:
				d := req.GetData()
				data := d.GetData()

				timeStamp := d.Start
				if doRelativeTime {
					if firstBlock == 0 {
						firstBlock = d.Start
					}
					timeStamp = timeStamp - firstBlock
				}

				record := make([]string, 2)
				for i, value := range data {
					record[0] = strconv.FormatFloat(float64(timeStamp)+(float64(1000000)/float64(d.Rate))*float64(i), 'f', -1, 64)
					record[1] = strconv.FormatFloat(float64(value), 'f', -1, 64)
					writer.Write(record)

				}

				break
			case api.GetTimeseriesRangeResponse_ERROR:
				err := req.GetError()
				log.Error(err.GetInfo())
			}
		}
	},
}

func init() {

	getCmd.Flags().BoolP("relative_time", "r",
		false, "Use relative time from start of channel in sec.")

}
