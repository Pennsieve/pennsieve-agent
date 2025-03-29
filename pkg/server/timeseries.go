package server

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	log "github.com/sirupsen/logrus"
	"slices"
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

	log.Info("GetTimeseriesRangeForChannels called - ", req.DatasetId, req.PackageId, req.RelativeTime)

	rangeChannel := make(chan models.TsBlock, 15)
	ctx := context.Background()

	tsService := s.TimeseriesService()

	// If channelIds not provided, then get all channels
	channels := req.ChannelIds
	minStartTime := 0

	response, err := tsService.GetChannelsForPackage(ctx, req.GetDatasetId(), req.GetPackageId(), req.Refresh)

	if len(channels) == 0 {
		log.Info("No Channels Supplied --> Returning all channels")
		for _, ch := range response {

			// Sending channel info to client.
			err = tsService.StreamChannelInfoToClient(ctx, ch, stream)
			if err != nil {
				log.Error("Error streaming channel info to client: ", err)
			}

			if minStartTime == 0 || minStartTime > int(ch.Start) {
				minStartTime = int(ch.Start)
			}

			channels = append(channels, ch.ChannelNodeId)
		}
	} else {
		for _, ch := range response {
			if slices.Contains(channels, ch.ChannelNodeId) {
				minStartTime = int(ch.Start)

				// Sending channel info to client.
				err = tsService.StreamChannelInfoToClient(ctx, ch, stream)
				if err != nil {
					log.Error("Error streaming channel info to client: ", err)
				}

			}
		}
	}

	var useStartTime, useEndTime uint64

	log.Debug("req time: ", req.StartTime, req.EndTime)

	if req.RelativeTime {
		useStartTime = uint64(req.StartTime*1000000) + uint64(minStartTime)
		useEndTime = uint64(req.EndTime*1000000) + uint64(minStartTime)
	} else {
		useStartTime = uint64(req.StartTime)
		useEndTime = uint64(req.EndTime)

	}
	log.Debug("use time: ", useStartTime, useEndTime)

	// Create WaitGroup: The waitgroup is canceled when all blocks are streamed to client
	var streamWg sync.WaitGroup
	streamWg.Add(1)

	// Use the service to get the blocks for range and put them on channel once available locally
	go func() {
		defer close(rangeChannel)

		// TODO: Update service to take a list of channels
		err := tsService.GetRangeBlocksForChannels(
			ctx, req.DatasetId, req.PackageId, channels, useStartTime, useEndTime, rangeChannel)
		if err != nil {
			log.Error("GetTimeseriesRangeForChannels err: ", err)
		}

	}()

	// Use the service to grab local blocks of data and Stream to client over gRPC
	go func() {
		tsService.StreamBlocksToClient(ctx, rangeChannel, useStartTime, useEndTime, stream)
		defer streamWg.Done()
	}()

	// Wait for WaitGroup to be Done to keep gRPC Stream open until all data is sent to client.
	streamWg.Wait()
	log.Debug("returning from function")

	return nil
}

func (s *agentServer) ResetCache(ctx context.Context, req *api.ResetCacheRequest) (*api.SimpleStatusResponse, error) {

	resetAll := true
	packageId := ""
	if req.Id != nil {
		resetAll = false
		packageId = req.GetId()
	}

	log.Info(fmt.Sprintf("%v - %s", resetAll, packageId))

	tsService := s.TimeseriesService()
	err := tsService.ResetCache(ctx, packageId, resetAll)
	if err != nil {
		log.Error("ResetCache err: ", err)
		return nil, err
	}

	response := api.SimpleStatusResponse{Status: "Success"}
	return &response, nil
}
