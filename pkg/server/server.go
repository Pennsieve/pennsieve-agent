// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/config"
	"github.com/pennsieve/pennsieve-agent/pkg/service"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"sync"
)

var GRPCServer *grpc.Server

var Version = "development"
var LogLevel = "INFO"

type server struct {
	v1.UnimplementedAgentServer
	subscribers sync.Map // subscribers is a concurrent map that holds mapping from a client ID to it's subscriber.
	cancelFncs  sync.Map // cancelFncs is a concurrent map that holds cancel functions for upload routines.

	client *pennsieve.Client

	Manifest *service.ManifestService
	User     *service.UserService
}

type uploadSession struct {
	manifestId int32
	cancelFnc  context.CancelFunc
}

type sub struct {
	stream   v1.Agent_SubscribeServer // stream is the server side of the RPC stream
	finished chan<- bool              // finished is used to signal closure of a client subscribing goroutine
}

// API ENDPOINT IMPLEMENTATIONS
// --------------------------------------------

// Subscribe handles a subscribe request from a client
func (s *server) Subscribe(request *v1.SubscribeRequest, stream v1.Agent_SubscribeServer) error {
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
func (s *server) Unsubscribe(ctx context.Context, request *v1.SubscribeRequest) (*v1.SubscribeResponse, error) {
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
	return &v1.SubscribeResponse{}, nil
}

func (s *server) Stop(ctx context.Context, request *v1.StopRequest) (*v1.StopResponse, error) {

	log.Println("Stopping Agent Server.")
	go GRPCServer.Stop()

	return &v1.StopResponse{Success: true}, nil
}

// Ping returns true and can be used to check if the agent is running
func (s *server) Ping(ctx context.Context, request *v1.PingRequest) (*v1.PingResponse, error) {

	return &v1.PingResponse{Success: true}, nil
}

// Version returns the version of the installed Pennsieve Agent and CLU
func (s *server) Version(ctx context.Context, request *v1.VersionRequest) (*v1.VersionResponse, error) {

	return &v1.VersionResponse{Version: Version, LogLevel: LogLevel}, nil
}

// HELPER FUNCTIONS
// ----------------------------------------------

// messageSubscribers sends a string message to all grpc-update subscribers and the log
func (s *server) messageSubscribers(message string) {

	// Send message to log
	log.Printf("SubscriberMessgae: %s", message)

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
		if err := sub.stream.Send(&v1.SubscribeResponse{
			Type: v1.SubscribeResponse_EVENT,
			MessageData: &v1.SubscribeResponse_EventInfo{
				EventInfo: &v1.SubscribeResponse_EventResponse{Details: message}},
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

	// Create new server
	GRPCServer = grpc.NewServer()
	server := &server{}

	db, _ := config.InitializeDB()
	manifestStore := store.NewManifestStore(db)
	manifestFileStore := store.NewManifestFileStore(db)
	server.Manifest = service.NewManifestService(manifestStore, manifestFileStore)

	userInfoStore := store.NewUserInfoStore(db)
	userSettingsStore := store.NewUserSettingsStore(db)
	server.User = service.NewUserService(userInfoStore, userSettingsStore)

	client, err := config.InitPennsieveClient(userSettingsStore, userInfoStore)
	if err != nil {
		return err
	}

	server.client = client
	server.Manifest.SetPennsieveClient(client)
	server.User.SetPennsieveClient(client)

	// Register services
	v1.RegisterAgentServer(GRPCServer, server)

	fmt.Printf("GRPC server listening on: %s", lis.Addr())

	if err := GRPCServer.Serve(lis); err != nil {
		fmt.Println("failed to serve: ", err)
		return err
	}

	return nil
}

func SetupLogger() {
	homedir, _ := os.UserHomeDir()
	logFilePath := homedir + "/.pennsieve/agent.log"
	_, err := os.Stat(logFilePath)

	// TODO: Set log level after moving to logrus

	logFileLocation, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0744)
	if err != nil {
		log.Fatalln(err)
	}
	log.SetOutput(logFileLocation)
}
