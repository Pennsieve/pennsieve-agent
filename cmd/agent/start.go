package agent

import (
	"fmt"
	gp "github.com/pennsieve/pennsieve-agent/pkg/agent"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
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
			command := exec.Command("pennsieve-agent", "agent", "start")

			//stdout, _ := command.StdoutPipe()

			//TODO: Remove when not testing
			err := command.Start()
			if err != nil {
				log.Fatalln(err)
			}

			////Check output of the external process to ensure server is running
			//for {
			//	tmp := make([]byte, 1024)
			//	_, err := stdout.Read(tmp)
			//	message := string(tmp)
			//	if strings.HasPrefix(message, "failed to listen") {
			//		log.Fatalln(message)
			//	} else if strings.HasPrefix(message, "failed to serve") {
			//		log.Fatalln(message)
			//	} else if strings.HasPrefix(message, "GRPC agent listening") {
			//		log.Println(message)
			//		break
			//	}
			//
			//	if err != nil {
			//		break
			//	}
			//}

			//stdout = nil

			// Check that We

			// Store server PID in lock file, so we can terminate server when needed.
			fmt.Printf("Agent start, [PID] %d running...\n", command.Process.Pid)
			ioutil.WriteFile("agent.lock", []byte(fmt.Sprintf("%d", command.Process.Pid)), 0666)
			daemon = false
			os.Exit(0)
		}

		fmt.Println("Running Agent NOT as daemon")
		err := gp.StartAgent()
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	startCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "is daemon?")
}
