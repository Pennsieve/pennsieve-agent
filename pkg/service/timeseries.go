package service

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	log "github.com/sirupsen/logrus"
)

type TimeseriesService interface {
	GetRangeBlocksForChannels(
		Ctx context.Context,
		DatasetNodeId string,
		PackageNodeId string,
		ChannelNodeIds []string,
		StartTime uint64,
		EndTime uint64,
		rangeChannel chan<- models.TsBlock,
	) error
	GetChannelsForPackage(
		Ctx context.Context,
		DatasetNodeId string,
		PackageNodeId string,
		Refresh bool,
	) ([]models.TsChannel, error)
	StreamBlocksToClient(
		ctx context.Context,
		blocks <-chan models.TsBlock,
		stream api.Agent_GetTimeseriesRangeForChannelsServer) error
}

type TimeseriesServiceImpl struct {
	tsStore    store.TimeseriesStore
	subscriber shared.Subscriber
	client     *pennsieve.Client
}

func NewTimeseriesService(ts store.TimeseriesStore, c *pennsieve.Client, s shared.Subscriber) TimeseriesService {
	return &TimeseriesServiceImpl{
		tsStore:    ts,
		client:     c,
		subscriber: s,
	}
}

// GetRangeBlocksForChannels retrieves
func (t *TimeseriesServiceImpl) GetRangeBlocksForChannels(
	ctx context.Context,
	DatasetNodeId string,
	PackagenodeId string,
	ChannelNodeIds []string,
	StartTime uint64,
	EndTime uint64,
	rangeChannel chan<- models.TsBlock,
) error {

	// Check which blocks are available on server
	result, err := t.client.Timeseries.GetRangeBlocks(ctx, DatasetNodeId, PackagenodeId, StartTime, EndTime, ChannelNodeIds)
	if err != nil {
		return err
	}

	// Check which Blocks are already cached on the local machine
	cachedBlocks, err := t.tsStore.GetRangeBlocksForChannels(ctx, ChannelNodeIds, StartTime, EndTime)
	if err != nil {
		return err
	}

	for _, ch := range result.Channels {
		for _, r := range ch.Ranges {

			isCached := false
			for _, cb := range cachedBlocks {
				if cb.BlockNodeId == r.ID {
					isCached = true
					break
				}
			}

			// If cached --> load data and send on stream
			// Else download data --> store in cache --> send on stream
			if isCached {

			} else {

				log.Info("Downloading")
				downloadImpl := shared.NewDownloader(t.subscriber, t.client)
				_, err := downloadImpl.DownloadFileFromPresignedUrl(ctx, r.PreSignedURL, r.ID, "1")
				if err != nil {
					log.Error("Error downloading file from presigned url: ", err)
					return err
				}

			}

			// TODO: Check if block in cache

			cRange := models.TsBlock{
				BlockNodeId:   r.ID,
				ChannelNodeId: ch.ChannelID,
				Rate:          float64(r.SamplingRate),
				Location:      r.PreSignedURL,
				StartTime:     r.StartTime,
				EndTime:       r.EndTime,
			}

			rangeChannel <- cRange
		}

	}
	// Check if blocks are already cached, and if not, fetch them from server

	return nil
}

func (t *TimeseriesServiceImpl) GetChannelsForPackage(
	ctx context.Context,
	datasetNodeId string,
	packageNodeId string,
	refresh bool,
) ([]models.TsChannel, error) {

	log.Debug("GetChannelsForPackage called")

	// Check if cached.
	cachedChannels, err := t.tsStore.GetChannelsForPackage(ctx, packageNodeId)
	if err != nil {
		return nil, err
	}

	// Try Fetch from server if empty or refresh flag is true
	if len(cachedChannels) == 0 || refresh {

		log.Debug("GetChannelsForPackage refresh called")

		// Use Pennsieve-go to fetch from server
		channels, err := t.client.Timeseries.GetChannels(ctx, datasetNodeId, packageNodeId)
		if err != nil {
			log.Errorf("Error getting channels for dataset node id %v: %v", datasetNodeId, err)
			return nil, err
		}

		// Map into agent structs
		var result []models.TsChannel
		for _, channel := range channels {
			result = append(result, models.TsChannel{
				ChannelNodeId: channel.ChannelID,
				PackageNodeId: packageNodeId,
				Name:          channel.Name,
				Start:         int64(channel.StartTime),
				End:           int64(channel.EndTime),
				Unit:          channel.Unit,
				Rate:          channel.Rate,
			})
		}

		// Save in SQL-database
		err = t.tsStore.StoreChannelsForPackage(ctx, datasetNodeId, result)
		if err != nil {
			log.Error("Error storing channels for dataset node id %v: %v", datasetNodeId, err)
			return nil, err
		}

		cachedChannels = result
	}

	return cachedChannels, nil
}

// StreamBlocksToClient streams blocks of data from local cache to the client application over gRPC
// It assumes that the blocks are locally available.
func (t *TimeseriesServiceImpl) StreamBlocksToClient(
	ctx context.Context,
	blocks <-chan models.TsBlock,
	stream api.Agent_GetTimeseriesRangeForChannelsServer) error {

	// TODO: This method should load and crop data to the specific start and end-time of request.

	for block := range blocks {
		log.Info(fmt.Sprintf("StreamBlocksToClient called - %s", block.BlockNodeId))
		stream.Send(&api.GetTimeseriesRangeResponse{
			Type: api.GetTimeseriesRangeResponse_RANGE_DATA,
			MessageData: &api.GetTimeseriesRangeResponse_Data{Data: &api.GetTimeseriesRangeResponse_RangeData{
				Start:     uint64(block.StartTime),
				End:       uint64(block.EndTime),
				Rate:      float32(block.Rate),
				ChannelId: block.ChannelNodeId,
				Data:      []float32{1, 2, 3, 4},
			},
			}})
	}

	return nil
}
