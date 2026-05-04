package shared

import "testing"

func TestIsSecretFile(t *testing.T) {
	cases := map[string]bool{
		// known credential basenames
		".netrc":          true,
		".pgpass":         true,
		".npmrc":          true,
		".pypirc":         true,
		".my.cnf":         true,
		".My.Cnf":         true, // case-insensitive
		// .env family
		".env":            true,
		".env.local":      true,
		".env.production": true,
		".env.test":       true,
		".ENV":            true,
		// SSH private keys
		"id_rsa":          true,
		"id_rsa.bak":      true,
		"id_ed25519":      true,
		"id_ecdsa":        true,
		"id_dsa":          true,
		"ID_RSA":          true,
		// false positives we want to avoid
		"environment":     false, // would falsely match .env without proper separator
		".envoy":          false, // would falsely match .env
		"id_rsa_notes":    false, // underscore != dot separator
		"data.env":        false, // .env not at start
		"netrc":           false, // missing leading dot
		"normal.txt":      false,
		".gitkeep":        false,
		".bidsignore":     false,
		"":                false,
	}
	for name, want := range cases {
		if got := IsSecretFile(name); got != want {
			t.Errorf("IsSecretFile(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsSecretDir(t *testing.T) {
	cases := map[string]bool{
		".ssh":         true,
		".SSH":         true,
		".aws":         true,
		".gnupg":       true,
		".kube":        true,
		".docker":      true,
		".azure":       true,
		".gcp":         true,
		"ssh":          false, // missing leading dot
		"data":         false,
		".git":         false,
		"node_modules": false,
	}
	for name, want := range cases {
		if got := IsSecretDir(name); got != want {
			t.Errorf("IsSecretDir(%q) = %v, want %v", name, got, want)
		}
	}
}