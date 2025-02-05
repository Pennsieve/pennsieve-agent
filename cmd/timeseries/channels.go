package timeseries

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/config"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
)

var ChannelsCmd = &cobra.Command{
	Use:   "channels <package_id>",
	Short: "Prints the channels for a given package",
	Long:  `This methods displays a list of the channels for a given package.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		package_id := args[0]

		doRefresh, err := cmd.Flags().GetBool("refresh_cache")
		if err != nil {
			log.Printf("Error: Cannot get flag \"refresh_cache\": %s", err)
			return
		}

		// Getting the active dataset
		db, _ := config.InitializeDB()
		userSettingsStore := store.NewUserSettingsStore(db)
		s, _ := userSettingsStore.Get()
		if len(s.UseDatasetId) == 0 {
			fmt.Println("\nError: No dataset specified; use 'pennsieve dataset use <node-id>' to set active dataset.")
			return
		}

		// Now open GRPC and request channels from server
		port := viper.GetString("agent.port")
		conn, err := grpc.NewClient(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))

		if err != nil {
			log.Println("Error connecting to GRPC Server: ", err)
			return
		}

		client := api.NewAgentClient(conn)

		response, err := client.GetTimeseriesChannels(context.Background(), &api.GetTimeseriesChannelsRequest{
			DatasetId: s.UseDatasetId,
			PackageId: package_id,
			Refresh:   doRefresh,
		})
		if err != nil {
			log.Println("Error retrieving channels: ", err)
			return
		}

		PrettyPrintList(response.Channel, package_id)

	},
}

func init() {
	ChannelsCmd.Flags().BoolP("refresh_cache", "r",
		false, "Should refresh cache of channels")
}

func PrettyPrintList(ch []*api.TimeseriesChannel, pkg string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle(fmt.Sprintf("Channels - %s", pkg))
	t.AppendHeader(table.Row{"NAME", "RATE", "UNIT", "NODE ID", "START TIME", "END TIME"})
	for _, d := range ch {
		t.AppendRow([]interface{}{d.Name, d.Rate, d.Unit, d.Id, d.StartTime, d.EndTime})
	}

	t.Render()
}
