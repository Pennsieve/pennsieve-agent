package projectpath

import (
	"path/filepath"
	"runtime"
)

var (
	_, b, _, _ = runtime.Caller(0)

	// Returns root folder of this project
	Root = filepath.Join(filepath.Dir(b), "../..")
)
