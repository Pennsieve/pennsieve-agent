package server

import (
	"context"
	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
)

type GRPCTestSuite struct {
	ServerTestSuite
	ctx        context.Context
	listener   *bufconn.Listener
	grpcServer *grpc.Server
	grpcClient pb.AgentClient
}

func (s *GRPCTestSuite) SetupTest() {
	s.ServerTestSuite.SetupTest()
	buffer := 1024 * 1024
	s.listener = bufconn.Listen(buffer)

	s.grpcServer = grpc.NewServer()
	pb.RegisterAgentServer(s.grpcServer, s.testServer)
	go func() {
		err := s.grpcServer.Serve(s.listener)
		if err != nil {
			s.FailNow("error starting GRPC server: ", err)
		}
	}()

	s.ctx = context.Background()
	conn, err := grpc.DialContext(s.ctx, "",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return s.listener.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		s.FailNow("error connecting to GRPC server: ", err)
	}

	s.grpcClient = pb.NewAgentClient(conn)
}

func (s *GRPCTestSuite) TearDownTest() {
	err := s.listener.Close()
	s.grpcServer.Stop()
	s.ServerTestSuite.TearDownTest()
	s.NoError(err)
}

func (s *GRPCTestSuite) TestPing() {
	req := pb.PingRequest{}
	resp, err := s.grpcClient.Ping(s.ctx, &req)
	if s.NoError(err) {
		s.True(resp.Success)
	}
}

func (s *GRPCTestSuite) TestVersion() {
	expectedVersion := "grpc-test"
	Version = expectedVersion
	req := pb.VersionRequest{}
	resp, err := s.grpcClient.Version(s.ctx, &req)
	if s.NoError(err) {
		s.Equal(expectedVersion, resp.Version)
		s.Equal(logrus.GetLevel().String(), resp.LogLevel)
	}
}

func TestGRPCSuite(t *testing.T) {
	suite.Run(t, new(GRPCTestSuite))
}
