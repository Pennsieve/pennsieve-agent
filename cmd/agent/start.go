package agent

import (
	"context"
	"fmt"
	gp "github.com/pennsieve/pennsieve-agent/pkg/server"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
	"os/exec"
)

var daemon bool
var startCmd = &cobra.Command{
	Use:   "start [Options]",
	Short: "Starts the Pennsieve Agent (blocking)",

	Run: func(cmd *cobra.Command, args []string) {

		// Allow parent to set daemon flag
		if len(args) > 0 && args[0] == "daemon" {
			daemon = true
		}

		port, _ := cmd.Flags().GetString("port")
		if len(port) == 0 {
			port = viper.GetString("agent.port")
		}

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)

		// Check if Pennsieve Server is running at the selected port
		resp, _ := client.Ping(context.Background(), &pb.PingRequest{})
		if resp != nil {
			fmt.Printf("Pennsieve Agent is already running on port: %s\n", viper.GetString("agent.port"))
			return
		}

		// Code example from: https://developpaper.com/start-and-stop-operations-of-golang-daemon/
		if daemon {
			command := exec.Command("pennsieve", "agent", "start")
			//command := exec.Command("go", "run", "main.go", "agent", "start")
			err := command.Start()
			if err != nil {
				log.Fatalln(err)
			}

			// Store server PID in lock file, so we can terminate server when needed.
			fmt.Printf("Pennsieve Agent started on port: %s\n", viper.GetString("agent.port"))
			daemon = false
			os.Exit(0)
		}

		fmt.Println("Running Agent NOT as daemon")
		err = gp.StartAgent()
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	startCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "is daemon?")
}
