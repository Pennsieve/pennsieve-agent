package agent

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/api"
	"github.com/pennsieve/pennsieve-go"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"net"
)

type server struct {
	UnimplementedAgentServer
}

// UploadPath recursively uploads a folder to the Pennsieve Platform.
func (s *server) UploadPath(req *UploadRequest, stream Agent_UploadPathServer) error {

	client := pennsieve.NewClient() // Create simple suninitialized client
	activeUser, err := api.GetActiveUser(client)
	if err != nil {
		fmt.Println(err)

	}
	fmt.Println(activeUser)

	apiToken := viper.GetString(activeUser.Profile + ".api_token")
	apiSecret := viper.GetString(activeUser.Profile + ".api_secret")
	client.Authentication.Authenticate(apiToken, apiSecret)

	if err != nil {
		fmt.Println("ERROR")
	}

	client.Authentication.GetAWSCredsForUser()

	uploadToAWS(*client, req.BasePath)

	resp := UploadStatus{
		Id:       "status 1",
		Progress: 10,
	}
	if err := stream.Send(&resp); err != nil {
		return err
	}

	return nil
}

// StartAgent initiates the local gRPC server and checks if it runs.
func StartAgent() error {

	// Get port for gRPC server
	port := viper.GetString("agent.port")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("failed to listen: ", err)
		return err
	}

	// Create new server
	grpcServer := grpc.NewServer()

	// Register services
	RegisterAgentServer(grpcServer, &server{})

	fmt.Println("GRPC agent listening on: ", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		fmt.Println("failed to serve: ", err)
		return err
	}

	return nil
}
