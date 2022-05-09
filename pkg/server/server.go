// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"fmt"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"sync"
)

type server struct {
	pb.UnimplementedAgentServer
	subscribers  sync.Map // subscribers is a concurrent map that holds mapping from a client ID to it's subscriber.
	cancelFncs   sync.Map // cancelFncs is a concurrent map that holds cancel functions for upload routines.
	uploadBucket string
}

type uploadSession struct {
	manifestId string
	cancelFnc  context.CancelFunc
}

type sub struct {
	stream   pb.Agent_SubscribeServer // stream is the server side of the RPC stream
	finished chan<- bool              // finished is used to signal closure of a client subscribing goroutine
}

// API ENDPOINT IMPLEMENTATIONS
// --------------------------------------------

// Subscribe handles a subscribe request from a client
func (s *server) Subscribe(request *pb.SubscribeRequest, stream pb.Agent_SubscribeServer) error {
	// Handle subscribe request
	log.Printf("Received subscribe request from ID: %d", request.Id)

	fin := make(chan bool)
	// Save the subscriber stream according to the given client ID
	s.subscribers.Store(request.Id, sub{stream: stream, finished: fin})

	ctx := stream.Context()
	// Keep this scope alive because once this scope exits - the stream is closed
	for {
		select {
		case <-fin:
			log.Printf("Closing stream for client ID: %d", request.Id)
			s.messageSubscribers(fmt.Sprintf("Closing stream for client ID: %d", request.Id))
			return nil
		case <-ctx.Done():
			log.Printf("Client ID %d has disconnected", request.Id)
			s.messageSubscribers(fmt.Sprintf("Closing stream for client ID: %d", request.Id))
			return nil
		}
	}
}

// Unsubscribe handles a unsubscribe request from a client
// Note: this function is not called but it here as an example of an unary RPC for unsubscribing clients
func (s *server) Unsubscribe(ctx context.Context, request *pb.SubscribeRequest) (*pb.SubsrcribeResponse, error) {
	v, ok := s.subscribers.Load(request.Id)
	if !ok {
		return nil, fmt.Errorf("failed to load subscriber key: %d", request.Id)
	}
	sub, ok := v.(sub)
	if !ok {
		return nil, fmt.Errorf("failed to cast subscriber value: %T", v)
	}
	select {
	case sub.finished <- true:
		log.Printf("Unsubscribed client: %d", request.Id)
	default:
		// Default case is to avoid blocking in case client has already unsubscribed
	}
	s.subscribers.Delete(request.Id)
	return &pb.SubsrcribeResponse{}, nil
}

// HELPER FUNCTIONS
// ----------------------------------------------

// messageSubscribers sends a string message to all grpc-update subscribers
func (s *server) messageSubscribers(message string) {
	// A list of clients to unsubscribe in case of error
	var unsubscribe []int32

	// Iterate over all subscribers and send data to each client
	s.subscribers.Range(func(k, v interface{}) bool {
		id, ok := k.(int32)
		if !ok {
			log.Printf("Failed to cast subscriber key: %T", k)
			return false
		}
		sub, ok := v.(sub)
		if !ok {
			log.Printf("Failed to cast subscriber value: %T", v)
			return false
		}
		// Send data over the gRPC stream to the client
		if err := sub.stream.Send(&pb.SubsrcribeResponse{
			Type: pb.SubsrcribeResponse_EVENT,
			MessageData: &pb.SubsrcribeResponse_EventInfo{
				EventInfo: &pb.SubsrcribeResponse_EventResponse{Details: message}},
		}); err != nil {
			log.Printf("Failed to send data to client: %v", err)
			select {
			case sub.finished <- true:
				log.Printf("Unsubscribed client: %d", id)
			default:
				// Default case is to avoid blocking in case client has already unsubscribed
			}
			// In case of error the client would re-subscribe so close the subscriber stream
			unsubscribe = append(unsubscribe, id)
		}
		return true
	})

	// Unsubscribe erroneous client streams
	for _, id := range unsubscribe {
		s.subscribers.Delete(id)
	}
}

// StartAgent initiates the local gRPC server and checks if it runs.
func StartAgent() error {

	//Setup Logger
	SetupLogger()

	// Get port for gRPC server
	port := viper.GetString("agent.port")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("failed to listen: ", err)
		return err
	}

	//// initialize logger
	//var cfg Config
	//cfg.LogLevel = 0
	//cfg.LogTimeFormat = "MM/DD/YY hh:mmAM/PM"

	// Create new server
	grpcServer := grpc.NewServer()
	server := &server{
		uploadBucket: "pennsieve-dev-test-new-upload",
	}

	// Register services
	pb.RegisterAgentServer(grpcServer, server)

	fmt.Printf("GRPC server listening on: %s", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		fmt.Println("failed to serve: ", err)
		return err
	}

	return nil
}

func SetupLogger() {

	homedir, _ := os.UserHomeDir()
	logFilePath := homedir + "/.pennsieve/agent.log"
	_, err := os.Stat(logFilePath)

	logFileLocation, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0744)
	if err != nil {
		log.Fatalln(err)
	}
	log.SetOutput(logFileLocation)
}
