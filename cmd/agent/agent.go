package agent

import (
	"github.com/spf13/cobra"
)

// AgentCmd represents the server command
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Starts the Pennsieve Agent",
	Long: `Start the Pennsieve agent as a background task. 

The Pennsieve agent is a daemon that runs in the background and handles certain tasks associated with 
interacting with the Pennsieve platform.

The agent runs a small server that exposes functionality that can be used by any of the Pennsieve clients 
(i.e. Python, Go, CLI, Javascript, etc.). The Pennsieve agent connects to a local database which is used to cache
information from the Pennsieve Platform and manages upload sessions. 

The agent also manages uploading processes. That is, clients use the agent to specify the upload manifest and 
subsequently initiate the upload session. Files are uploaded by the agent in the background. Users can get status 
updates from the agent by running the 'pennsieve agent subscribe' method. This will open a channel through which the
agent sends status updates to the client.

Use 'pennsieve agent stop' to stop a running Pennsieve agent. 

Use 'pennsieve agent start' to run the agent as a blocking process.
`,

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
