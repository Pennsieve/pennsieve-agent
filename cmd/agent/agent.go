package agent

import (
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Starts the Agent gRPC server",

	Run: func(cmd *cobra.Command, args []string) {

		args = append(args, "daemon")
		startCmd.Run(cmd, args)

	},
}

func init() {
	AgentCmd.AddCommand(startCmd)
	AgentCmd.AddCommand(stopCmd)
	AgentCmd.AddCommand(subscribeCmd)
}
