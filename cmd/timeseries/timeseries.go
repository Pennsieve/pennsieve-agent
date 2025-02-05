package timeseries

import (
	"github.com/spf13/cobra"
)

var TimeseriesCmd = &cobra.Command{
	Use:   "timeseries [command] [...Args]",
	Short: "Interact with timeseries on Pennsieve",
	Long: `
  You can download packages and datasets using one of the 
  respective subcommands. Files will be downloaded by the 
  agent in the background and you can check progress by running 
  the agent subscriber.`,
}

func init() {
	TimeseriesCmd.AddCommand(ChannelsCmd)
	TimeseriesCmd.AddCommand(getCmd)
}
