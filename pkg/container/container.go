package container

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"net"
	"os"
)

type ServerContainer interface {
	StartAgent() error
}

type agentServerContainer struct {
	grpcServer v1.AgentServer
}

func NewAgentServerContainer() ServerContainer {
	return &agentServerContainer{}
}

// StartAgent initiates the local gRPC server and checks if it runs.
func (c *agentServerContainer) StartAgent() error {

	//Setup Logger
	SetupLogger()

	// Get port for gRPC server
	port := viper.GetString("agent.port")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("failed to listen: ", err)
		return err
	}

	// Register services
	GRPCServer := grpc.NewServer()
	serverImplementation, _ := server.NewAgentServer(GRPCServer)
	v1.RegisterAgentServer(GRPCServer, serverImplementation)

	fmt.Printf("GRPC server listening on: %s", lis.Addr())

	if err := GRPCServer.Serve(lis); err != nil {
		fmt.Println("failed to serve: ", err)
		return err
	}

	return nil
}

func SetupLogger() {

	log.SetFormatter(&log.JSONFormatter{})
	ll, err := log.ParseLevel(os.Getenv("PENNSIEVE_LOG_LEVEL"))
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(ll)
	}

	homedir, _ := os.UserHomeDir()

	// Ensure folder is created
	os.MkdirAll(homedir+"/.pennsieve", os.ModePerm)

	logFilePath := homedir + "/.pennsieve/agent.log"
	_, err = os.Stat(logFilePath)

	logFileLocation, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0744)
	if err != nil {
		log.Fatalln(err)
	}
	log.SetOutput(logFileLocation)
}
