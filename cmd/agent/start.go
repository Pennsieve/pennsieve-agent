package agent

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/container"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"os"
	"os/exec"
	"time"
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
			log.Error("Error connecting to GRPC Server: ", err)
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)

		// Check if Pennsieve Server is running at the selected port
		resp, _ := client.Ping(context.Background(), &api.PingRequest{})
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

			// Wait 2 seconds to allow agent to start in separate process
			time.Sleep(2 * time.Second)

			// Check if agent is running
			conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				fmt.Println("Error connecting to GRPC Server: ", err)
				return
			}
			defer conn.Close()

			client := api.NewAgentClient(conn)

			// Check if Pennsieve Server is running at the selected port
			_, err = client.Ping(context.Background(), &api.PingRequest{})
			if err != nil {
				st := status.Convert(err)
				switch st.Code() {
				case codes.Unavailable:
					fmt.Println("Unknown error while starting Pennsieve Agent Server. \n" +
						"Please check the agent.log file for more details.")
				default:
					fmt.Println("Unknown error while starting Pennsieve Agent Server. \n" +
						"Please check the agent.log file for more details.")
				}
				os.Exit(1)
			} else {

				fmt.Printf("Pennsieve Agent started on port: %s\n", viper.GetString("agent.port"))
				daemon = false
				os.Exit(0)
			}

		}

		fmt.Println("Running Agent NOT as daemon")
		grpcContainer := container.NewAgentServerContainer()
		err = grpcContainer.StartAgent()
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	startCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "is daemon?")
}
