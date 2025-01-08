package service

import (
	"context"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
)

type TimeseriesService interface {
	GetRangeBlocksForChannel(
		Ctx context.Context,
		OrganizationNodeId int64,
		DatasetNodeId string,
		ChannelNodeId string,
		StartTime uint64,
		EndTime uint64,
	) (*models.ChannelWithRanges, error)
}
