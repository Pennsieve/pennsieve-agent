package server

import (
	"context"
	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	log "github.com/sirupsen/logrus"
)

func (s *agentServer) Stop(ctx context.Context, request *pb.StopRequest) (*pb.StopResponse, error) {

	log.Info("Stopping Agent Server.")
	go s.grpcServer.Stop()

	return &pb.StopResponse{Success: true}, nil
}

// Ping returns true and can be used to check if the agent is running
func (s *agentServer) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {

	return &pb.PingResponse{Success: true}, nil
}

// Version returns the version of the installed Pennsieve Agent and CLU
func (s *agentServer) Version(ctx context.Context, request *pb.VersionRequest) (*pb.VersionResponse, error) {

	return &pb.VersionResponse{Version: Version, LogLevel: log.GetLevel().String()}, nil
}
