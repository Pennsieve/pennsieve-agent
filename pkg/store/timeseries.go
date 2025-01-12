package store

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	"strings"
)

type TimeseriesStore interface {
	GetChannelsForPackage(
		Ctx context.Context,
		PackageNodeId string,
	) ([]models.TimeSeriesChannel, error)
	GetRangeBlocksForChannels(
		Ctx context.Context,
		ChannelNodeIds []string,
		StartTime uint64,
		EndTime uint64,
	) ([]models.TimeSeriesContinuousRange, error)
}

func NewTimeseriesStore(db *sql.DB) TimeseriesStore {
	return &timeseriesStore{
		db: db,
	}
}

type timeseriesStore struct {
	db *sql.DB
}

func (s *timeseriesStore) GetChannelsForPackage(
	ctx context.Context,
	packageNodeId string,
) ([]models.TimeSeriesChannel, error) {

	statement, err := s.db.PrepareContext(ctx, `SELECT id, node_id, package_id, name, start_time, end_time, unit, rate FROM ts_channel WHERE package_id = $1`)
	if err != nil {
		return nil, err
	}

	defer statement.Close()
	rows, err := statement.Query(packageNodeId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.TimeSeriesChannel
	for rows.Next() {
		var channel models.TimeSeriesChannel
		err := rows.Scan(
			&channel.ID,
			&channel.NodeId,
			&channel.PackageNodeId,
			&channel.Name,
			&channel.Start,
			&channel.End,
			&channel.Unit,
			&channel.Rate,
		)
		if err != nil {
			return nil, err
		}

		channels = append(channels, channel)
	}

	return channels, nil
}

func (s *timeseriesStore) GetRangeBlocksForChannels(
	ctx context.Context,
	channelNodeIds []string,
	startTime uint64,
	endTime uint64,
) ([]models.TimeSeriesContinuousRange, error) {

	channelIdString := fmt.Sprintf("%s", strings.Join(channelNodeIds, ", "))

	statement, err := s.db.PrepareContext(ctx, `
		SELECT id, node_id, channel_node_id, location, start_time, end_time
		FROM ts_range
		WHERE ((start_time <= $1 AND end_time > $1)
		   OR (start_time >= $1 AND end_time <= $2)
		   OR (start_time <$2 AND end_time > $2)
		   OR (start_time <=$1 AND end_time >= $2))
		   AND channel_node_id IN($3)`)

	defer statement.Close()

	if err != nil {
		return nil, err
	}
	fmt.Println(startTime, endTime, channelIdString)

	rows, err := statement.Query(startTime, endTime, channelIdString)
	if err != nil {
		fmt.Println("helo")
		return nil, err
	}
	defer rows.Close()

	var ranges []models.TimeSeriesContinuousRange
	for rows.Next() {
		var rng models.TimeSeriesContinuousRange
		err := rows.Scan(
			&rng.ID,
			&rng.NodeId,
			&rng.Channel,
			&rng.Location,
			&rng.StartTime,
			&rng.EndTime,
		)
		if err != nil {
			return nil, err
		}

		ranges = append(ranges, rng)
	}

	return ranges, nil
}
