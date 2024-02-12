// Package shared helpers.go: For short helper functions that may find use across packages
package shared

import (
	"strings"
)

// GetLeafDirectory Return the leaf directory from a file path
func GetLeafDirectory(inputPath string, osFilePathSeparator string) string {

	tokens := strings.Split(inputPath, osFilePathSeparator)
	numTokens := len(tokens)

	return tokens[numTokens-1]
}
