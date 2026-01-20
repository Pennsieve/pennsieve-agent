package service

import (
	"compress/gzip"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

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
		startTime uint64,
		endTime uint64,
		stream api.Agent_GetTimeseriesRangeForChannelsServer) error
	StreamChannelInfoToClient(
		ctx context.Context,
		channel models.TsChannel,
		stream api.Agent_GetTimeseriesRangeForChannelsServer) error
	ResetCache(
		ctx context.Context,
		packagenodeId string,
		resetAll bool,
	) error
}

type TimeseriesServiceImpl struct {
	tsStore       store.TimeseriesStore
	subscriber    shared.Subscriber
	client        *pennsieve.Client
	cacheLocation string
}

func NewTimeseriesService(ts store.TimeseriesStore, c *pennsieve.Client, s shared.Subscriber) TimeseriesService {
	homedir, _ := os.UserHomeDir()

	// Ensure cache folder is created
	os.MkdirAll(filepath.Join(homedir, ".pennsieve", "timeseries"), os.ModePerm)

	return &TimeseriesServiceImpl{
		tsStore:       ts,
		client:        c,
		subscriber:    s,
		cacheLocation: filepath.Join(homedir, ".pennsieve", "timeseries"),
	}
}

func (t *TimeseriesServiceImpl) ResetCache(ctx context.Context, packageNodeId string, resetAll bool) error {

	var packageIds []string
	var err error
	if resetAll {
		packageIds, err = t.tsStore.GetCachedPackageIds(ctx)
		if err != nil {
			return err
		}
	} else {
		packageIds = append(packageIds, packageNodeId)
	}

	log.Info("PackageIds: ", packageIds)

	for _, packageId := range packageIds {

		blocks, err := t.tsStore.GetLocalBlocksForPackage(ctx, packageId)
		if err != nil {
			log.Error("Failed to get local blocks for package", packageId)
			return err
		}

		log.Info("blocks: ", blocks)

		// Remove files from disk
		for _, block := range blocks {
			log.Info("remove block: ", block)
			err := os.Remove(block.Location)
			if err != nil {
				log.Error("Failed to remove block", block.Location)
				continue
			}

		}

		// Remove blocks from database
		log.Info("Removing blocks for package: ", packageId)
		err = t.tsStore.RemoveBlocksForPackage(ctx, packageId)
		if err != nil {
			log.Error("Failed to remove blocks for package", packageId)
		}

		// Remove
	}

	return nil
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
	log.Info(fmt.Sprintf("d: %s, p: %s", DatasetNodeId, PackagenodeId))
	log.Info(fmt.Sprintf("Channel Node Ids: %v", ChannelNodeIds))

	start := time.Now()

	// Get ranges for all channels
	result, err := t.client.Timeseries.GetRangeBlocks(ctx, DatasetNodeId, PackagenodeId, StartTime, EndTime, "")
	if err != nil {
		log.Error(err)
		return err
	}

	//// Check which Blocks are already cached on the local machine
	//cachedBlocks, err := t.tsStore.GetRangeBlocksForChannels(ctx, []string{ChannelNodeId}, StartTime, EndTime)
	//if err != nil {
	//	return err
	//}

	log.Printf("Binomial took %s", time.Since(start))

	for _, ch := range result.Channels {

		// Only process the channels that were requested --> discard others
		// TODO: Refactor such that timeseries service can take a list of channels to return urls for
		if !slices.Contains(ChannelNodeIds, ch.ChannelID) {
			continue
		}

		// Check which Blocks are already cached on the local machine
		cachedBlocks, err := t.tsStore.GetRangeBlocksForChannels(ctx, []string{ch.ChannelID}, StartTime, EndTime)
		if err != nil {
			return err
		}

		log.Info("Cached Blocks: ", cachedBlocks)

		// Sort ranges by StartTime to ensure correct ordering
		sort.Slice(ch.Ranges, func(i, j int) bool {
			return ch.Ranges[i].StartTime < ch.Ranges[j].StartTime
		})

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
			targetLocation := filepath.Join(t.cacheLocation, r.ID)
			if isCached {
				log.Info("Getting Cached Value")

			} else {

				log.Info("Downloading")
				downloadImpl := shared.NewDownloader(t.subscriber, t.client)
				//targetLocation = filepath.Join(t.cacheLocation, r.ID)
				_, err := downloadImpl.DownloadFileFromPresignedUrl(ctx, r.PreSignedURL, targetLocation, "1")
				if err != nil {
					log.Error("Error downloading file from presigned url: ", err)
					return err
				}
				// Store in db
				err = t.tsStore.StoreBlockForChannel(ctx, r.ID, ch.ChannelID, targetLocation, uint64(r.StartTime), uint64(r.EndTime))
				if err != nil {
					log.Error(err)
				}

			}

			cRange := models.TsBlock{
				BlockNodeId:   r.ID,
				ChannelNodeId: ch.ChannelID,
				Rate:          float64(r.SamplingRate),
				Location:      targetLocation,
				StartTime:     r.StartTime,
				EndTime:       r.EndTime,
			}

			// Sending block to channel.
			// The channel should receive blocks in the correct order by channel.
			// Channel 1 - block 1 : Channel 1 - block 2 : Channel 2 - block 1 : Channel 2 - block2
			rangeChannel <- cRange
		}

	}

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
			log.Error(fmt.Sprintf("Error storing channels for dataset node id %v: %v", datasetNodeId, err))
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
	startTime uint64,
	endTime uint64,
	stream api.Agent_GetTimeseriesRangeForChannelsServer) error {

	for block := range blocks {
		log.Info(fmt.Sprintf("StreamBlocksToClient called - %s", block.BlockNodeId))

		fileContents, err := readGzFile(block.Location)
		if err != nil {
			fileContents, err = os.ReadFile(block.Location)
			log.Error(err)
		}

		// Crop block if needed
		var croppedSlice []byte
		intBlockStart := uint64(block.StartTime)
		intBlockEnd := uint64(block.EndTime)

		croppedStart := intBlockStart
		croppedEnd := intBlockEnd
		if intBlockStart < startTime && intBlockEnd < endTime {
			// Crop from beginning
			cropIndex := 8 * (int64(float64((startTime-intBlockStart)/1000000)*block.Rate*8) / 8)
			log.Info(fmt.Sprintf("1START %d -- %d - %d", cropIndex, intBlockEnd-intBlockStart, len(fileContents)))

			croppedSlice = fileContents[cropIndex:]
			croppedStart = startTime

		} else if intBlockStart < startTime && intBlockEnd > endTime {
			// Crop from beginning and end
			cropIndex1 := 8 * (int64(float64((startTime-intBlockStart)/1000000)*block.Rate*8) / 8)
			cropIndex2 := 8 * (int64((float64(endTime-intBlockStart)/1000000)*block.Rate*8) / 8)
			log.Info(fmt.Sprintf("2START %d - %d -- %d - %d", cropIndex1, cropIndex2, intBlockEnd-intBlockStart, len(fileContents)))
			croppedSlice = fileContents[cropIndex1:cropIndex2]
			log.Info(fmt.Sprintf("cropped slice %d", len(croppedSlice)))

			croppedStart = startTime
			croppedEnd = endTime

		} else if intBlockEnd > endTime {
			// Crop from end
			log.Info(fmt.Sprintf("3START %d - %d -- %d - %d", intBlockStart, intBlockEnd, intBlockEnd-intBlockStart, len(fileContents)))
			cropIndex := 8 * (int64((float64(endTime-intBlockStart)/1000000)*block.Rate*8) / 8)
			croppedSlice = fileContents[:cropIndex]

			croppedEnd = endTime

			log.Info(fmt.Sprintf("3CROP END +%d - %d", len(croppedSlice), cropIndex))
		} else {
			log.Info("4CROP ALL")
			croppedSlice = fileContents
		}

		data := BytesToFloat32s(croppedSlice)
		if data == nil {
			log.Error("Error converting cropped slice")
		}

		err = stream.Send(&api.GetTimeseriesRangeResponse{
			Type: api.GetTimeseriesRangeResponse_RANGE_DATA,
			MessageData: &api.GetTimeseriesRangeResponse_Data{Data: &api.GetTimeseriesRangeResponse_RangeData{
				Start:     croppedStart,
				End:       croppedEnd,
				Rate:      float32(block.Rate),
				ChannelId: block.ChannelNodeId,
				Data:      data,
			},
			}})
		if err != nil {
			log.Error(err)
			return err
		}
	}

	return nil
}

func (t *TimeseriesServiceImpl) StreamChannelInfoToClient(
	ctx context.Context,
	ch models.TsChannel,
	stream api.Agent_GetTimeseriesRangeForChannelsServer) error {

	err := stream.Send(&api.GetTimeseriesRangeResponse{
		Type: api.GetTimeseriesRangeResponse_CHANNEL_INFO,
		MessageData: &api.GetTimeseriesRangeResponse_Channel{Channel: &api.GetTimeseriesRangeResponse_ChannelInfo{
			ChannelId: ch.ChannelNodeId,
			Name:      ch.Name,
			Unit:      ch.Unit,
			Rate:      float32(ch.Rate),
		}}})
	if err != nil {
		return err
	}
	return nil
}

// readGzFile returns an array of bytes from the gzipped file.
func readGzFile(filename string) ([]byte, error) {
	fi, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	fz, err := gzip.NewReader(fi)
	if err != nil {
		return nil, err
	}
	defer fz.Close()

	s, err := io.ReadAll(fz)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// BytesToFloat32s converts a byte array to a float32 array.
// It assumes the values in the byte array are 64-bit bigEndian floats.
func BytesToFloat32s(data []byte) []float32 {
	if len(data)%8 != 0 {
		// A float64 is 8 bytes, so the input byte array must be a multiple of 8
		return nil
	}

	float32s := make([]float32, len(data)/8)
	for i := range len(float32s) {
		bits := binary.BigEndian.Uint64(data[i*8 : (i+1)*8])
		float32s[i] = float32(math.Float64frombits(bits))
	}
	return float32s
}
