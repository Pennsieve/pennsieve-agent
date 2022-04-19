package agent

import (
	"fmt"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"net"
)

type server struct {
	UnimplementedAgentServer
}

func (s *server) UploadPath(req *UploadRequest, stream Agent_UploadPathServer) error {

	resp := UploadStatus{
		Id:       "status 1",
		Progress: 10,
	}
	if err := stream.Send(&resp); err != nil {
		return err
	}

	return nil
}

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
