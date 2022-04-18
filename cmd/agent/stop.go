package agent

import (
	"github.com/spf13/cobra"
	"io/ioutil"
	"os/exec"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Agent",
	Run: func(cmd *cobra.Command, args []string) {
		strb, _ := ioutil.ReadFile("agent.lock")
		command := exec.Command("kill", string(strb))
		command.Start()
		println("Agent stopped.")
	},
}
