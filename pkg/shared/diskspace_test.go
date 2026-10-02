package shared

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreeBytesOfAFolderThatDoesNotExistYet(t *testing.T) {
	root := t.TempDir()
	want, err := FreeBytes(root)
	require.NoError(t, err)
	require.Greater(t, want, uint64(0))

	got, err := FreeBytes(filepath.Join(root, "new", "deeper"))
	require.NoError(t, err)
	assert.InDelta(t, float64(want), float64(got), float64(want)/100, "the parent's disk")
}

func TestHumanBytes(t *testing.T) {
	assert.Equal(t, "999 B", HumanBytes(999))
	assert.Equal(t, "4.0 MB", HumanBytes(4024644))
	assert.Equal(t, "1.6 TB", HumanBytes(1575648896014))
}
