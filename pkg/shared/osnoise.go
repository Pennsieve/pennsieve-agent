package shared

import "strings"

// OS-generated metadata files that get added to user folders without explicit
// consent and are essentially never legitimate research data. We skip them at
// manifest-add time so users don't accidentally ship them to a dataset.
//
// Match is on the file/directory basename, case-insensitively (Windows uses
// case-insensitive filesystems and naming conventions vary).
var osNoiseFiles = map[string]struct{}{
	".ds_store":   {}, // macOS Finder metadata
	".localized":  {}, // macOS Finder localization marker
	"thumbs.db":   {}, // Windows Explorer thumbnail cache
	"desktop.ini": {}, // Windows folder customization
}

// Directories whose entire contents are OS noise — skipped wholesale rather
// than walked into.
var osNoiseDirs = map[string]struct{}{
	".spotlight-v100": {}, // macOS Spotlight index
	".trashes":        {}, // macOS trash on external volumes
	".trash":          {}, // generic trash
	".fseventsd":      {}, // macOS file system events
	".appledouble":    {}, // macOS resource forks on non-HFS volumes
	".appledb":        {},
	".appledesktop":   {},
	"__macosx":        {}, // macOS metadata folder created in zip archives
	"$recycle.bin":    {}, // Windows recycle bin
}

// IsOSNoiseFile reports whether the given file basename is OS-generated
// noise that should be filtered out.
func IsOSNoiseFile(basename string) bool {
	if _, ok := osNoiseFiles[strings.ToLower(basename)]; ok {
		return true
	}
	// AppleDouble forks (._foo) are macOS-on-non-HFS shadow files containing
	// resource-fork metadata. They're never the user's actual content.
	if strings.HasPrefix(basename, "._") {
		return true
	}
	return false
}

// IsOSNoiseDir reports whether the given directory basename should be
// skipped entirely (the whole subtree).
func IsOSNoiseDir(basename string) bool {
	_, ok := osNoiseDirs[strings.ToLower(basename)]
	return ok
}