package agent

import (
	"github.com/spf13/cobra"
	"io/ioutil"
	"os/exec"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Agent",
	Long:  `Stops the Pennsieve agent if it is running in the background.`,
	Run: func(cmd *cobra.Command, args []string) {
		strb, _ := ioutil.ReadFile("server.lock")
		command := exec.Command("kill", string(strb))
		command.Start()
		println("Agent stopped.")
	},
}
