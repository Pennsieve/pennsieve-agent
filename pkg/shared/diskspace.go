package shared

import (
	"fmt"
	"os"
	"path/filepath"
)

// FreeBytes is the space available to the user on the disk of path, or of
// its nearest existing parent when path doesn't exist yet.
func FreeBytes(path string) (uint64, error) {
	path = filepath.Clean(path)
	for {
		if _, err := os.Stat(path); err == nil {
			break
		}
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}
	return freeBytes(path)
}

// HumanBytes formats a size in decimal units, as the web app does.
func HumanBytes(n int64) string {
	units := []string{"B", "kB", "MB", "GB", "TB", "PB"}
	v, i := float64(n), 0
	for v >= 1000 && i < len(units)-1 {
		v /= 1000
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}
