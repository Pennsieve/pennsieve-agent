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

The Pennsieve Agent can be configured using a config.ini file or by using environment variables. You can set the 
following ENV Variables:

1. PENNSIEVE_API_KEY				The api-key for the user
2. PENNSIEVE_API_SECRET				The api-secret for the user
3. PENNSIEVE_AGENT_PORT				The port on which to run the agent (default: 9000)
4. PENNSIEVE_AGENT_CHUNK_SIZE 		The size in MB per chunk while uploading (default: 32)
5. PENNSIEVE_AGENT_UPLOAD_WORKERS	The number of parallel upload processes (default: 5)


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
