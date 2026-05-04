package shared

import "testing"

func TestIsOSNoiseFile(t *testing.T) {
	cases := map[string]bool{
		".DS_Store":      true,
		".ds_store":      true,
		".localized":     true,
		"Thumbs.db":      true,
		"thumbs.db":      true,
		"Desktop.ini":    true,
		"._hidden":       true,
		"._.DS_Store":    true,
		"data.csv":       false,
		"DS_Store":       false, // missing leading dot
		"my.localized":   false, // not exact basename
		"normal_file":    false,
		".gitkeep":       false, // legitimate placeholder
		"__init__.py":    false,
		"":               false,
	}
	for name, want := range cases {
		if got := IsOSNoiseFile(name); got != want {
			t.Errorf("IsOSNoiseFile(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsOSNoiseDir(t *testing.T) {
	cases := map[string]bool{
		".Spotlight-V100": true,
		".spotlight-v100": true,
		".Trashes":        true,
		".fseventsd":      true,
		".AppleDouble":    true,
		"__MACOSX":        true,
		"$RECYCLE.BIN":    true,
		"$recycle.bin":    true,
		"data":            false,
		".git":            false, // not OS noise; user may want it
		"node_modules":    false, // not OS-level
	}
	for name, want := range cases {
		if got := IsOSNoiseDir(name); got != want {
			t.Errorf("IsOSNoiseDir(%q) = %v, want %v", name, got, want)
		}
	}
}