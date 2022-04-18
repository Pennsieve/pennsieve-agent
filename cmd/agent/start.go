package agent

import (
	"fmt"
	pb "github.com/pennsieve/pennsieve-agent/agent"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"net"
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
			command.Start()
			fmt.Printf("Agent start, [PID] %d running...\n", command.Process.Pid)
			ioutil.WriteFile("agent.lock", []byte(fmt.Sprintf("%d", command.Process.Pid)), 0666)
			daemon = false
			os.Exit(0)
		} else {
			fmt.Println("agent start")
		}
		startAgent()

	},
}

func init() {
	startCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "is daemon?")

}

func startAgent() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register services
	pb.RegisterAgentServer(grpcServer, &Server{})

	log.Printf("GRPC agent listening on %v", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
