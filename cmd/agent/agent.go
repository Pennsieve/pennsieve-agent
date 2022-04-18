package agent

import (
	pb "github.com/pennsieve/pennsieve-agent/agent"
	"github.com/spf13/cobra"
)

const (
	port = ":9000"
)

type Server struct {
	pb.UnimplementedAgentServer
}

// serverCmd represents the agent command
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Starts the Agent gRPC agent",

	Run: func(cmd *cobra.Command, args []string) {

		args = append(args, "daemon")
		startCmd.Run(cmd, args)

	},
}

func init() {
	AgentCmd.AddCommand(startCmd)
	AgentCmd.AddCommand(stopCmd)

}
