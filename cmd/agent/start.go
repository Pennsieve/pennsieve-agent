package agent

import (
	"fmt"
	gp "github.com/pennsieve/pennsieve-agent/agent"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

var daemon bool
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the Schema gRPC agent",

	Run: func(cmd *cobra.Command, args []string) {

		// Allow parent to set daemon flag
		if len(args) > 0 && args[0] == "daemon" {
			daemon = true
		}

		// Code example from: https://developpaper.com/start-and-stop-operations-of-golang-daemon/
		if daemon {
			fmt.Println("daemon")
			command := exec.Command("pennsieve-agent", "agent", "start")
			stdout, _ := command.StdoutPipe()
			command.Stderr = command.Stdout
			err := command.Start()
			fmt.Println(err)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			for {
				tmp := make([]byte, 1024)
				_, err := stdout.Read(tmp)
				message := string(tmp)
				if strings.HasPrefix(message, "failed to listen") {
					fmt.Print(message)
					os.Exit(1)
				} else if strings.HasPrefix(message, "failed to serve") {
					fmt.Print(message)
					os.Exit(1)
				} else if strings.HasPrefix(message, "GRPC agent listening") {
					break
				}
				fmt.Print(message)
				if err != nil {
					break
				}
			}

			fmt.Printf("Agent start, [PID] %d running...\n", command.Process.Pid)
			ioutil.WriteFile("agent.lock", []byte(fmt.Sprintf("%d", command.Process.Pid)), 0666)
			daemon = false
			os.Exit(0)
		} else {
			fmt.Println("agent start")
		}
		err := gp.StartAgent()
		fmt.Println("Error: ", err)
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	startCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "is daemon?")
}
