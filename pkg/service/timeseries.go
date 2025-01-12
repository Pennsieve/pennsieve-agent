package service

import (
	"context"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
)

type TimeseriesService interface {
	GetRangeBlocksForChannels(
		Ctx context.Context,
		OrganizationNodeId int64,
		DatasetNodeId string,
		ChannelNodeIds []string,
		StartTime uint64,
		EndTime uint64,
	) (*models.ChannelWithRanges, error)
}

type TimeseriesServiceImpl struct {
	tsStore store.TimeseriesStore
	client  *pennsieve.Client
}

func NewTimeseriesService(ts store.TimeseriesStore, c *pennsieve.Client) TimeseriesService {
	return &TimeseriesServiceImpl{
		tsStore: ts,
		client:  c,
	}
}

func (t *TimeseriesServiceImpl) GetRangeBlocksForChannels(
	ctx context.Context,
	OrganizationNodeId int64,
	DatasetNodeId string,
	ChannelNodeIds []string,
	StartTime uint64,
	EndTime uint64,
) (*models.ChannelWithRanges, error) {

	return nil, nil
}
