package store

import (
	"context"
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
