package agent

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Agent",
	Long:  `Stops the Pennsieve agent if it is running in the background.`,
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalln("Unable to get home-folder.")
		}
		strb, _ := ioutil.ReadFile(fmt.Sprintf("%s/agent.lock", filepath.Join(home, ".pennsieve")))
		command := exec.Command("kill", string(strb))
		command.Start()
		println("Agent stopped.")
	},
}
