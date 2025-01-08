// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/config"
	"github.com/pennsieve/pennsieve-agent/pkg/service"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var GRPCServer *grpc.Server

var Version = "development"

type server struct {
	pb.UnimplementedAgentServer
	subscribers        sync.Map // subscribers is a concurrent map that holds mapping from a client ID to it's subscriber.
	cancelFncs         sync.Map // cancelFncs is a concurrent map that holds cancel functions for upload routines.
	syncCancelFncs     sync.Map // syncCancelFncs is a map that hold synctimers for each active dataset.
	downloadCancelFncs sync.Map // downloadCancelFncs is a map that holds cancel functions for download routines.

	client *pennsieve.Client

	Manifest *service.ManifestService
	User     *service.UserService
	Account  *service.AccountService
}

type uploadSession struct {
	manifestId int32
	cancelFnc  context.CancelFunc
}

type downloadSession struct {
	id        string
	cancelFnc context.CancelFunc
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
	log.Info("Received subscribe request from ID: ", request.Id)

	fin := make(chan bool)
	// Save the subscriber stream according to the given client ID
	s.subscribers.Store(request.Id, sub{stream: stream, finished: fin})

	ctx := stream.Context()
	// Keep this scope alive because once this scope exits - the stream is closed
	for {
		select {
		case <-fin:
			log.Info(fmt.Sprintf("Closing stream for client ID: %d", request.Id))
			s.messageSubscribers(fmt.Sprintf("Closing stream for client ID: %d", request.Id))
			return nil
		case <-ctx.Done():
			log.Info(fmt.Sprintf("Client ID %d has disconnected", request.Id))
			s.messageSubscribers(fmt.Sprintf("Closing stream for client ID: %d", request.Id))
			return nil
		}
	}
}

// Unsubscribe handles a unsubscribe request from a client
// Note: this function is not called but it here as an example of an unary RPC for unsubscribing clients
func (s *server) Unsubscribe(ctx context.Context, request *pb.SubscribeRequest) (*pb.SubscribeResponse, error) {
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
		log.Info(fmt.Sprintf("Unsubscribed client: %d", request.Id))
	default:
		// Default case is to avoid blocking in case client has already unsubscribed
	}
	s.subscribers.Delete(request.Id)
	return &pb.SubscribeResponse{}, nil
}

func (s *server) Stop(ctx context.Context, request *pb.StopRequest) (*pb.StopResponse, error) {

	log.Info("Stopping Agent Server.")
	go GRPCServer.Stop()

	return &pb.StopResponse{Success: true}, nil
}

// Ping returns true and can be used to check if the agent is running
func (s *server) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {

	return &pb.PingResponse{Success: true}, nil
}

// Version returns the version of the installed Pennsieve Agent and CLU
func (s *server) Version(ctx context.Context, request *pb.VersionRequest) (*pb.VersionResponse, error) {

	return &pb.VersionResponse{Version: Version, LogLevel: log.GetLevel().String()}, nil
}

// HELPER FUNCTIONS
// ----------------------------------------------

func (s *server) stopSyncTimers() {
	s.syncCancelFncs.Range(func(key interface{}, value interface{}) bool {
		fmt.Println("STOP SYNCING on ", key.(int32))
		tmr := value.(chan struct{})
		tmr <- struct{}{}
		return true
	})
}

// messageSubscribers sends a string message to all grpc-update subscribers and the log
func (s *server) messageSubscribers(message string) {

	// Send message to log
	log.Info("SubscriberMessage: ", message)

	// A list of clients to unsubscribe in case of error
	var unsubscribe []int32

	// Iterate over all subscribers and send data to each client
	s.subscribers.Range(func(k, v interface{}) bool {
		id, ok := k.(int32)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber key: %T", k))
			return false
		}
		sub, ok := v.(sub)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber value: %T", v))
			return false
		}
		// Send data over the gRPC stream to the client
		if err := sub.stream.Send(&pb.SubscribeResponse{
			Type: pb.SubscribeResponse_EVENT,
			MessageData: &pb.SubscribeResponse_EventInfo{
				EventInfo: &pb.SubscribeResponse_EventResponse{Details: message}},
		}); err != nil {
			log.Warn(fmt.Sprintf("Failed to send data to client: %v", err))
			select {
			case sub.finished <- true:
				log.Info(fmt.Sprintf("Unsubscribed client: %d", id))
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

	db, err := config.InitializeDB()

	// Run Migrations if needed
	dbPath := viper.GetString("agent.db_path")
	m, err := migrate.New(
		"file://db/migrations",
		fmt.Sprintf("sqlite3://%s?_foreign_keys=on&mode=rwc&_journal_mode=WAL", dbPath),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("No change in database schema: ", err)
		} else {
			log.Fatal(err)
		}
	}

	if err != nil {
		fmt.Println("Error initializing DB --", err)
	}
	server, err := newServer(db)
	if err != nil {
		return err
	}
	// Register services
	pb.RegisterAgentServer(GRPCServer, server)

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

func newServer(db *sql.DB) (*server, error) {
	manifestStore := store.NewManifestStore(db)
	manifestFileStore := store.NewManifestFileStore(db)

	userInfoStore := store.NewUserInfoStore(db)
	userSettingsStore := store.NewUserSettingsStore(db)

	client, err := config.InitPennsieveClient(userSettingsStore, userInfoStore)
	if err != nil {
		return &server{}, err
	}
	server := server{}
	server.Manifest = service.NewManifestService(manifestStore, manifestFileStore)
	server.User = service.NewUserService(userInfoStore, userSettingsStore)

	server.client = client
	server.Manifest.SetPennsieveClient(client)
	server.User.SetPennsieveClient(client)

	server.Account = service.NewAccountService(client)

	return &server, nil
}
