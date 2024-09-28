package shared

import (
    "github.com/stretchr/testify/assert"
    "testing"
)

func TestCrCCheck(t *testing.T) {

    // Check CRC
    nrBytes := 1048576 // 1MB
    crc, err := GetFileCrc32("../../resources/test/.pennsieve/manifest.json", nrBytes)
    assert.NoError(t, err)
    assert.Equal(t, uint32(0xdb58346b), crc)

    // Check that crc is idempotent
    crc, err = GetFileCrc32("../../resources/test/.pennsieve/manifest.json", nrBytes)
    assert.NoError(t, err)
    assert.Equal(t, uint32(0xdb58346b), crc)

    // Check crc with maxbuffer that is smaller than file
    crc, err = GetFileCrc32("../../resources/test/.pennsieve/manifest.json", 50)
    assert.NoError(t, err)
    assert.Equal(t, uint32(0x46c16dc9), crc)
}