// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"fmt"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	log "github.com/sirupsen/logrus"
)

var Version = "development"

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
func (s *agentServer) Subscribe(request *pb.SubscribeRequest, stream pb.Agent_SubscribeServer) error {
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
func (s *agentServer) Unsubscribe(ctx context.Context, request *pb.SubscribeRequest) (*pb.SubscribeResponse, error) {
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
