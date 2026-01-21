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
    GetCachedPackageIds(
        Ctx context.Context,
    ) ([]string, error)
    GetLocalBlocksForPackage(
        Ctx context.Context,
        PackageNodeId string,
    ) ([]models.TsBlock, error)
    StoreBlockForChannel(
        Ctx context.Context,
        BlockNodeId string,
        ChannelNodeId string,
        Location string,
        StartTime uint64,
        EndTime uint64,
    ) error
    RemoveBlocksForPackage(
        ctx context.Context,
        packageId string,
    ) (err error)
}

func NewTimeseriesStore(db *sql.DB) TimeseriesStore {
    return &timeseriesStore{
        db: db,
    }
}

type timeseriesStore struct {
    db *sql.DB
}

func (s *timeseriesStore) GetCachedPackageIds(ctx context.Context) (ids []string, err error) {

    sqlStr := `SELECT DISTINCT package_id FROM ts_channel`

    rows, err := s.db.Query(sqlStr)
    if err != nil {
        return nil, err
    }

    var packageIds []string
    for rows.Next() {
        var packageId *string
        err := rows.Scan(&packageId)
        if err != nil {
            return nil, err
        }

        packageIds = append(packageIds, *packageId)
    }

    return packageIds, nil
}

func (s *timeseriesStore) GetLocalBlocksForPackage(ctx context.Context, packageId string) (blocks []models.TsBlock, err error) {

    sqlStr := `SELECT rng.node_id, rng.channel_node_id, rng.location, rng.start_time, rng.end_time, ts.package_id
    FROM ts_range AS rng JOIN ts_channel AS ts ON rng.channel_node_id == ts.node_id WHERE package_id = ?`

    stmt, err := s.db.PrepareContext(ctx, sqlStr)
    if err != nil {
        log.Error("Failed to prepare statement: ", err)
        return nil, err
    }
    defer stmt.Close()

    rows, err := stmt.Query(packageId)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var ranges []models.TsBlock
    for rows.Next() {
        var rng models.TsBlock
        var pckId *string
        err := rows.Scan(
            &rng.BlockNodeId,
            &rng.ChannelNodeId,
            &rng.Location,
            &rng.StartTime,
            &rng.EndTime,
            &pckId,
        )
        if err != nil {
            return nil, err
        }

        ranges = append(ranges, rng)
    }

    return ranges, nil
}

func (s *timeseriesStore) RemoveBlocksForPackage(ctx context.Context, packageId string) (err error) {

    sqlStr := "DELETE FROM ts_range WHERE channel_node_id IN (SELECT DISTINCT channel_node_id FROM ts_channel JOIN ts_range ON ts_range.channel_node_id = ts_channel.node_id WHERE package_id = ?)"

    _, err = s.db.Exec(sqlStr, packageId)
    if err != nil {
        return err
    }

    return nil

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

    log.Info(ChannelNodeId)
    _, err = stmt.ExecContext(ctx, BlockNodeId, ChannelNodeId, location, StartTime, EndTime)
    if err != nil {
        log.Error("Failed to store blocks for package: ", err)
        return err
    }

    return nil
}

func (s *timeseriesStore) StoreChannelsForPackage(
    ctx context.Context,
    PackageNodeId string,
    Channels []models.TsChannel) error {

    sqlStr := `INSERT INTO ts_channel(node_id, package_id, name, start_time, end_time, unit, rate) VALUES `
    var vals []interface{}

    for _, channel := range Channels {
        sqlStr += "(?,?,?,?,?,?,?),"
        vals = append(vals, channel.ChannelNodeId, channel.PackageNodeId,
            channel.Name, channel.Start, channel.End, channel.Unit, channel.Rate)
    }

    sqlStr = strings.TrimSuffix(sqlStr, ",")

    // Add ON CONFLICT clause to update existing rows instead of replacing
    sqlStr += ` ON CONFLICT(node_id) DO UPDATE SET 
		package_id = excluded.package_id,
		name = excluded.name,
		start_time = excluded.start_time,
		end_time = excluded.end_time,
		unit = excluded.unit,
		rate = excluded.rate`

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
