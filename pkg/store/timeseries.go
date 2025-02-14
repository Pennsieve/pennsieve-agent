package store

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	log "github.com/sirupsen/logrus"
	"strings"
)

type TimeseriesStore interface {
	StoreChannelsForPackage(
		Ctx context.Context,
		PackageNodeId string,
		Channels []models.TsChannel) error
	GetChannelsForPackage(
		Ctx context.Context,
		PackageNodeId string,
	) ([]models.TsChannel, error)
	GetRangeBlocksForChannels(
		Ctx context.Context,
		ChannelNodeIds []string,
		StartTime uint64,
		EndTime uint64,
	) ([]models.TsBlock, error)
	StoreBlockForChannel(
		Ctx context.Context,
		BlockNodeId string,
		ChannelNodeId string,
		Location string,
		StartTime uint64,
		EndTime uint64,
	) error
}

func NewTimeseriesStore(db *sql.DB) TimeseriesStore {
	return &timeseriesStore{
		db: db,
	}
}

type timeseriesStore struct {
	db *sql.DB
}

func (s *timeseriesStore) StoreBlockForChannel(ctx context.Context, BlockNodeId string, ChannelNodeId string, location string,
	StartTime uint64, EndTime uint64) error {

	sqlStr := "REPLACE INTO ts_range(node_id,channel_node_id,location,start_time,end_time) VALUES (?,?,?,?,?)"

	stmt, err := s.db.PrepareContext(ctx, sqlStr)
	if err != nil {
		log.Error("Failed to prepare statement: ", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, BlockNodeId, ChannelNodeId, location, StartTime, EndTime)
	if err != nil {
		log.Error("Failed to store channels for package: ", err)
		return err
	}

	return nil
}

func (s *timeseriesStore) StoreChannelsForPackage(
	ctx context.Context,
	PackageNodeId string,
	Channels []models.TsChannel) error {

	sqlStr := "REPLACE INTO ts_channel(node_id,package_id,name,start_time,end_time,unit,rate) VALUES "
	var vals []interface{}

	for _, channel := range Channels {
		sqlStr += "(?,?,?,?,?,?,?),"
		vals = append(vals, channel.ChannelNodeId, channel.PackageNodeId,
			channel.Name, channel.Start, channel.End, channel.Unit, channel.Rate)
	}

	sqlStr = strings.TrimSuffix(sqlStr, ",")

	stmt, err := s.db.PrepareContext(ctx, sqlStr)
	if err != nil {
		log.Error("Failed to prepare statement: ", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, vals...)
	if err != nil {
		log.Error("Failed to store channels for package: ", err)
		return err
	}

	log.Info("Successfully stored channels for package: ", PackageNodeId)

	return nil

}

func (s *timeseriesStore) GetChannelsForPackage(
	ctx context.Context,
	packageNodeId string,
) ([]models.TsChannel, error) {

	statement, err := s.db.PrepareContext(ctx, `SELECT node_id, package_id, name, start_time, end_time, unit, rate FROM ts_channel WHERE package_id = $1`)
	if err != nil {
		return nil, err
	}

	defer statement.Close()
	rows, err := statement.Query(packageNodeId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.TsChannel
	for rows.Next() {
		var channel models.TsChannel
		err := rows.Scan(
			&channel.ChannelNodeId,
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
) ([]models.TsBlock, error) {

	channelIdString := ""
	if len(channelNodeIds) > 0 {
		channelIdString = fmt.Sprintf("'%s'", strings.Join(channelNodeIds, "', '"))
	}

	log.Info(channelIdString)

	// Note: not ideal to use sprintf to get channelIds in there but
	// adding this in statement.query doesn't work.
	statement, err := s.db.PrepareContext(ctx, fmt.Sprintf(`
		SELECT node_id, channel_node_id, location, start_time, end_time
		FROM ts_range
		WHERE ((start_time <= $1 AND end_time > $1)
		   OR (start_time >= $1 AND end_time <= $2)
		   OR (start_time <$2 AND end_time > $2)
		   OR (start_time <=$1 AND end_time >= $2))
		   AND channel_node_id IN( %s )`, channelIdString))

	defer statement.Close()

	if err != nil {
		return nil, err
	}

	rows, err := statement.Query(startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ranges []models.TsBlock
	for rows.Next() {
		var rng models.TsBlock
		err := rows.Scan(
			&rng.BlockNodeId,
			&rng.ChannelNodeId,
			&rng.Location,
			&rng.StartTime,
			&rng.EndTime,
		)
		if err != nil {
			return nil, err
		}

		ranges = append(ranges, rng)
	}

	log.Info("Successfully fetched ranges: ", ranges)

	return ranges, nil
}

func convertToInterfaceSlice(stringSlice []string) []interface{} {
	interfaceSlice := make([]interface{}, len(stringSlice))
	for i, str := range stringSlice {
		interfaceSlice[i] = str
	}
	return interfaceSlice
}
