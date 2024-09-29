// Package shared helpers.go: For short helper functions that may find use across packages
package shared

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Add Helper functions here as required

// GetAbsolutePath returns an absolute path based on a partial patial path provided
func GetAbsolutePath(folderStr string) (string, error) {

	folder := path.Clean(folderStr) // removing '/' adn '//' and '.' and '..'
	if !strings.HasPrefix(folder, string(os.PathSeparator)) {

		// If the user provides an absolute path, then use that,
		// else check if we should use home folder or current folder as the prefix

		var err error
		var prefix string
		if strings.HasPrefix(folder, "~") {
			prefix, err = os.UserHomeDir()
		} else {
			prefix, err = os.Getwd()

		}

		if err != nil {
			fmt.Println(err)
			return "", err
		}

		folder = filepath.Join(prefix, folder)
	}

	return folder, nil
}
