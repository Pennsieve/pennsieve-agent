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
	"time"
)

var resetCmd = &cobra.Command{
	Use:   "reset [package-id]",
	Short: "Removes locally cached timeseries blocks",
	Long: `This function removes time-series blocks that are locally cached. If users provide 
a package-id, then only the blocks for the specific package are removed.If no arguments are 
provided, all cached blocks will be removed. Data will not be removed from the Pennsieve 
platform, and subsequent requests for data will force re-downloading the data from the 
Pennsieve servers.
`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		var packageId *string
		if len(args) > 0 {
			packageId = &args[0]
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
		_, err = client.ResetCache(ctx, &api.ResetCacheRequest{Id: packageId})
		if err != nil {
			log.Error("Error resetting cache: ", err)
			return
		}

		fmt.Println("Successfully reset cache")

	},
}

func init() {
}
