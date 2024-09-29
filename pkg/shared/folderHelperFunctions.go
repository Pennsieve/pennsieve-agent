package shared

import (
    log "github.com/sirupsen/logrus"
    "hash/crc32"
    "os"
)

func PathIsDirectory(path string) bool {
    result := false
    // get file info for path
    fi, err := os.Stat(path)
    if err != nil {
        log.Fatal("Error in checking whether path is a directory: ", err)
    } else {
        // check file info mode to determine if path is a directory or a file
        switch mode := fi.Mode(); {
        case mode.IsDir():
            result = true
        case mode.IsRegular():
            result = false
        }
    }
    return result
}

func GetFileCrc32(path string, maxBytes int) (uint32, error) {

    f, err := os.Open(path)
    defer f.Close()
    if err != nil {
        return 0, err
    }

    info, err := f.Stat()
    if err != nil {
        return 0, err
    }

    // Create buffer of specific size and read into buffer
    b1 := make([]byte, max(info.Size(), int64(maxBytes)))
    _, err = f.Read(b1)

    return crc32.ChecksumIEEE(b1), nil

}