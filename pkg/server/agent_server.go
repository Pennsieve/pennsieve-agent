package server

import (
	"database/sql"
	"fmt"
	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/config"
	"github.com/pennsieve/pennsieve-agent/pkg/service"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"sync"
)

type DependencyContainer interface {
	SqliteDB() *sql.DB
	TimeseriesStore() store.TimeseriesStore
	TimeseriesService() service.TimeseriesService
	PennsieveClient() (*pennsieve.Client, error)
	UserSettingsStore() store.UserSettingsStore
	UserInfoStore() store.UserInfoStore
	ManifestStore() store.ManifestStore
	ManifestService() *service.ManifestService
	UserService() *service.UserService
	AccountService() *service.AccountService
	messageSubscribers(message string)
	GetSubscribers() sync.Map
}

type agentServer struct {
	pb.UnimplementedAgentServer
	subscribers    sync.Map // subscribers is a concurrent map that holds mapping from a client ID to it's subscriber.
	cancelFncs     sync.Map // cancelFncs is a concurrent map that holds cancel functions for upload routines.
	syncCancelFncs sync.Map // syncCancelFncs is a map that hold synctimers for each active dataset.

	grpcServer *grpc.Server
	client     *pennsieve.Client
	sqliteDB   *sql.DB

	timeseriesStore   store.TimeseriesStore
	userInfoStore     store.UserInfoStore
	userSettingsStore store.UserSettingsStore
	manifestStore     store.ManifestStore
	manifestFileStore store.ManifestFileStore

	manifest          *service.ManifestService
	user              *service.UserService
	account           *service.AccountService
	timeseriesService service.TimeseriesService
}

func NewAgentServer(s *grpc.Server) (*agentServer, error) {
	return &agentServer{grpcServer: s}, nil
}

func (s *agentServer) GetSubscribers() sync.Map {
	return s.subscribers
}

func (s *agentServer) ManifestStore() store.ManifestStore {
	if s.manifestStore == nil {
		st := store.NewManifestStore(s.SqliteDB())
		s.manifestStore = st
	}

	return s.manifestStore
}
func (s *agentServer) ManifestFileStore() store.ManifestFileStore {
	if s.manifestFileStore == nil {
		st := store.NewManifestFileStore(s.SqliteDB())
		s.manifestFileStore = st
	}

	return s.manifestFileStore
}
func (s *agentServer) UserSettingsStore() store.UserSettingsStore {
	if s.userSettingsStore == nil {
		st := store.NewUserSettingsStore(s.SqliteDB())
		s.userSettingsStore = st
	}

	return s.userSettingsStore
}
func (s *agentServer) UserInfoStore() store.UserInfoStore {
	if s.userInfoStore == nil {
		st := store.NewUserInfoStore(s.SqliteDB())
		s.userInfoStore = st
	}

	return s.userInfoStore
}
func (s *agentServer) TimeseriesStore() store.TimeseriesStore {
	if s.timeseriesStore == nil {
		s.timeseriesStore = store.NewTimeseriesStore(s.SqliteDB())
	}
	return s.timeseriesStore
}
func (s *agentServer) PennsieveClient() (*pennsieve.Client, error) {
	if s.client == nil {
		client, err := config.InitPennsieveClient(s.UserSettingsStore(), s.UserInfoStore())
		if err != nil {
			return nil, err
		}
		s.client = client
	}

	return s.client, nil
}
func (s *agentServer) SqliteDB() *sql.DB {
	if s.sqliteDB == nil {
		db, err := config.InitializeDB()
		if err != nil {
			log.Fatal(err)
		}
		s.sqliteDB = db
	}
	return s.sqliteDB
}
func (s *agentServer) AccountService() *service.AccountService {
	if s.account == nil {
		client, _ := s.PennsieveClient()
		s.account = service.NewAccountService(
			client,
		)
	}

	return s.account
}
func (s *agentServer) UserService() *service.UserService {
	if s.user == nil {
		client, err := s.PennsieveClient()
		if err != nil {
			log.Error(err)
		}

		s.user = service.NewUserService(
			s.UserInfoStore(),
			s.UserSettingsStore(),
			client,
		)
	}

	return s.user
}
func (s *agentServer) ManifestService() *service.ManifestService {
	if s.manifest == nil {
		client, err := s.PennsieveClient()
		if err != nil {
			log.Error(err)
		}

		s.manifest = service.NewManifestService(
			s.ManifestStore(),
			s.ManifestFileStore(),
			client,
		)
	}

	return s.manifest
}
func (s *agentServer) TimeseriesService() service.TimeseriesService {
	if s.timeseriesService == nil {
		client, err := s.PennsieveClient()
		if err != nil {
			log.Error(err)
		}

		s.timeseriesService = service.NewTimeseriesService(
			s.TimeseriesStore(),
			client,
			s,
		)
	}

	return s.timeseriesService
}

// messageSubscribers sends a string message to all grpc-update subscribers and the log
func (s *agentServer) messageSubscribers(message string) {

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
		sub, ok := v.(shared.Sub)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber value: %T", v))
			return false
		}
		// Send data over the gRPC Stream to the client
		if err := sub.Stream.Send(&pb.SubscribeResponse{
			Type: pb.SubscribeResponse_EVENT,
			MessageData: &pb.SubscribeResponse_EventInfo{
				EventInfo: &pb.SubscribeResponse_EventResponse{Details: message}},
		}); err != nil {
			log.Warn(fmt.Sprintf("Failed to send data to client: %v", err))
			select {
			case sub.Finished <- true:
				log.Info(fmt.Sprintf("Unsubscribed client: %d", id))
			default:
				// Default case is to avoid blocking in case client has already unsubscribed
			}
			// In case of error the client would re-subscribe so close the subscriber Stream
			unsubscribe = append(unsubscribe, id)
		}
		return true
	})

	// Unsubscribe erroneous client streams
	for _, id := range unsubscribe {
		s.subscribers.Delete(id)
	}
}
