package shared

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCrCCheck(t *testing.T) {

	// Check CRC
	nrBytes := 1048576 // 1MB
	crc, err := GetFileCrc32("../../resources/test/pullTest/.pennsieve/manifest.json", nrBytes)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0xa1dc9897), crc)

	// Check that crc is idempotent
	crc, err = GetFileCrc32("../../resources/test/pulltest/.pennsieve/manifest.json", nrBytes)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0xa1dc9897), crc)

	// Check crc with maxbuffer that is smaller than file
	crc, err = GetFileCrc32("../../resources/test/pullTest/.pennsieve/manifest.json", 50)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0x3842f4f6), crc)
}
