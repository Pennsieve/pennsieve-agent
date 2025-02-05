package server

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	log "github.com/sirupsen/logrus"
	"sync"
)

func (s *agentServer) GetTimeseriesChannels(ctx context.Context, req *api.GetTimeseriesChannelsRequest) (*api.GetTimeseriesChannelsResponse, error) {

	log.Info(fmt.Sprintf("GetTimeseriesChannels called %s - %s", req.DatasetId, req.PackageId))
	channels, err := s.TimeseriesService().GetChannelsForPackage(ctx, req.DatasetId, req.PackageId, req.Refresh)
	if err != nil {
		return nil, err
	}

	var response []*api.TimeseriesChannel
	for _, channel := range channels {
		ch := api.TimeseriesChannel{
			Id:        channel.ChannelNodeId,
			Name:      channel.Name,
			StartTime: uint64(channel.Start),
			EndTime:   uint64(channel.End),
			Unit:      channel.Unit,
			Rate:      float32(channel.Rate),
		}

		response = append(response, &ch)
	}

	return &api.GetTimeseriesChannelsResponse{Channel: response}, nil
}

func (s *agentServer) GetTimeseriesRangeForChannels(req *api.GetTimeseriesRangeRequest, stream api.Agent_GetTimeseriesRangeForChannelsServer) error {

	log.Info("GetTimeseriesRangeForChannels called - ", req.DatasetId, req.PackageId)

	rangeChannel := make(chan models.TsBlock, 5)
	ctx := context.Background()

	tsService := s.TimeseriesService()

	// Create WaitGroup: The waitgroup is canceled when all blocks are streamed to client
	var streamWg sync.WaitGroup
	streamWg.Add(1)

	// Use the service to get the blocks for range and put them on channel once available locally
	go func() {
		err := tsService.GetRangeBlocksForChannels(
			ctx, req.DatasetId, req.PackageId, req.ChannelId, req.StartTime, req.EndTime, rangeChannel)
		if err != nil {
			log.Error("GetTimeseriesRangeForChannels err: ", err)
		}
		defer close(rangeChannel)
	}()

	// Use the service to grab local blocks of data and Stream to client over gRPC
	go func() {
		tsService.StreamBlocksToClient(ctx, rangeChannel, stream)
		defer streamWg.Done()
	}()

	// Wait for WaitGroup to be Done to keep gRPC Stream open until all data is sent to client.
	streamWg.Wait()
	log.Info("returning from function")

	return nil
}
