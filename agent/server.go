package agent

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/api"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/pennsieve/pennsieve-go"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"
)

type Config struct {
	LogLevel      int
	LogTimeFormat string
}

type server struct {
	pb.UnimplementedAgentServer
	subscribers sync.Map           // subscribers is a concurrent map that holds mapping from a client ID to it's subscriber.
	cancelFnc   context.CancelFunc // cancelFnc is a function that cancels the upload context.
}

type sub struct {
	stream   pb.Agent_SubscribeServer // stream is the server side of the RPC stream
	finished chan<- bool              // finished is used to signal closure of a client subscribing goroutine
}

// UploadPath recursively uploads a folder to the Pennsieve Platform.
func (s *server) UploadPath(ctx context.Context, request *pb.UploadRequest) (*pb.UploadResponse, error) {

	// On runtime panic, log the stacktrace but keep server alive
	defer func() {
		if x := recover(); x != nil {
			// recovering from a panic; x contains whatever was passed to panic()
			log.Printf("Run time panic: %v", x)
			log.Printf("Stacktrace: \n %s", string(debug.Stack()))
		}
	}()

	client := pennsieve.NewClient() // Create simple suninitialized client
	activeUser, err := api.GetActiveUser(client)
	if err != nil {
		fmt.Println(err)

	}

	apiToken := viper.GetString(activeUser.Profile + ".api_token")
	apiSecret := viper.GetString(activeUser.Profile + ".api_secret")
	client.Authentication.Authenticate(apiToken, apiSecret)

	if err != nil {
		fmt.Println("ERROR")
	}

	client.Authentication.GetAWSCredsForUser()

	err = s.uploadToAWS(*client, request.BasePath)

	log.Println("Returned from uploader")
	response := pb.UploadResponse{Status: "Upload completed."}
	return &response, nil
}

func (s *server) CreateManifest(ctx context.Context, request *pb.CreateManifestRequest) (*pb.CreateManifestResponse, error) {

	// 1. Get new Upload Session ID from Pennsieve Server

	//TODO replace with real call to server
	//uploadSessionID := uuid.New()

	// 2. Walk over folder and populate DB with file-paths.
	localPath := request.BasePath
	walker := make(fileWalk)
	go func() {
		// Gather the files to upload by walking the path recursively
		if err := filepath.WalkDir(localPath, walker.Walk); err != nil {
			log.Println("Walk failed:", err)
		}
		close(walker)
	}()

}

// CancelUpload cancels an ongoing upload.
func (s *server) CancelUpload(ctx context.Context, request *pb.CancelRequest) (*pb.CancelResponse, error) {
	s.cancelFnc()
	return &pb.CancelResponse{
		Status: "Success"}, nil
}

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
			messageSubscribers(s, fmt.Sprintf("Closing stream for client ID: %d", request.Id))
			return nil
		case <-ctx.Done():
			log.Printf("Client ID %d has disconnected", request.Id)
			messageSubscribers(s, fmt.Sprintf("Closing stream for client ID: %d", request.Id))
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

func (s *server) mockDataGenerator() {
	log.Printf("Starting data generation")
	for {
		time.Sleep(time.Second)

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
				Type:        0,
				MessageData: &pb.SubsrcribeResponse_Data{Data: fmt.Sprintf("data mock for: %d", id)},
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

	// initialize logger
	var cfg Config
	cfg.LogLevel = 0
	cfg.LogTimeFormat = "MM/DD/YY hh:mmAM/PM"

	// Create new server
	grpcServer := grpc.NewServer()
	server := &server{}

	// Register services
	pb.RegisterAgentServer(grpcServer, server)

	fmt.Printf("GRPC agent listening on: %s", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		fmt.Println("failed to serve: ", err)
		return err
	}

	return nil
}

func SetupLogger() {
	logFileLocation, _ := os.OpenFile("./test.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0744)
	log.SetOutput(logFileLocation)
}
