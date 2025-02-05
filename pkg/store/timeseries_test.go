package store

import (
	"context"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTimeseriesStore(t *testing.T) {

	packageNodeId := "N:package:1"

	store := NewTimeseriesStore(db)
	channels, err := store.GetChannelsForPackage(context.Background(), packageNodeId)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(channels))

}

func TestCreateChannels(t *testing.T) {

	store := NewTimeseriesStore(db)

	packageNodeId := "N:package:store-test"
	channels := []models.TimeSeriesChannel{
		{
			NodeId:        "N:channel:store-test-1",
			PackageNodeId: packageNodeId,
			Name:          "channel 1",
			Start:         1,
			End:           500,
			Unit:          "uV",
			Rate:          120.5,
		},
		{
			NodeId:        "N:channel:store-test-2",
			PackageNodeId: packageNodeId,
			Name:          "channel 2",
			Start:         50,
			End:           500,
			Unit:          "uV",
			Rate:          120.5,
		},
	}

	err := store.StoreChannelsForPackage(context.Background(), packageNodeId, channels)
	assert.NoError(t, err)

	results, err := store.GetChannelsForPackage(context.Background(), packageNodeId)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(results))
	assert.Equal(t, "N:channel:store-test-1", results[0].NodeId)

}

func TestRangeStore(t *testing.T) {

	store := NewTimeseriesStore(db)

	// Ranges in database are [1-100), [100-150)
	ranges, err := store.GetRangeBlocksForChannels(context.Background(), []string{"N:channel:1"}, 50, 150)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ranges))

	ranges, err = store.GetRangeBlocksForChannels(context.Background(), []string{"N:channel:1"}, 1, 100)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ranges))

	ranges, err = store.GetRangeBlocksForChannels(context.Background(), []string{"N:channel:1"}, 0, 50)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ranges))
	assert.Equal(t, "19-1", ranges[0].NodeId)

	ranges, err = store.GetRangeBlocksForChannels(context.Background(), []string{"N:channel:1"}, 110, 200)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ranges))
	assert.Equal(t, "19-2", ranges[0].NodeId)

	ranges, err = store.GetRangeBlocksForChannels(context.Background(), []string{"N:channel:1"}, 0, 200)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ranges))

}
