package shared

import "strings"

// Conservative list of basenames known to carry credentials, API tokens, or
// private keys. Matched on basename only (case-insensitive) to avoid false
// positives from research data with paths that incidentally contain "ssh"
// or "aws" — silent skip with a loud log entry is safer than refusing to
// upload, and far safer than the alternative of leaking a credential into
// a research dataset.
//
// Match is intentionally narrow. We'd rather miss an obscure secret file
// (recoverable: user notices, re-adds explicitly) than skip a legitimate
// research file the user actually wanted (annoying and trust-eroding).

var secretFiles = map[string]struct{}{
	".netrc":  {}, // HTTP/FTP credentials
	".pgpass": {}, // Postgres passwords
	".npmrc":  {}, // may contain auth tokens
	".pypirc": {}, // PyPI auth tokens
	".my.cnf": {}, // MySQL credentials
}

// secretFilePrefixes match basenames that start with these strings followed
// either by end-of-string or a dot. Catches .env / .env.local / .env.prod
// and SSH private-key naming conventions (id_rsa, id_rsa.bak, id_ed25519).
var secretFilePrefixes = []string{
	".env",
	"id_rsa",
	"id_ed25519",
	"id_ecdsa",
	"id_dsa",
}

// secretDirs are directories that conventionally hold credentials/keys; we
// skip the whole subtree.
var secretDirs = map[string]struct{}{
	".ssh":    {},
	".aws":    {},
	".gnupg":  {},
	".kube":   {},
	".docker": {},
	".azure":  {},
	".gcp":    {},
}

// IsSecretFile reports whether the given basename matches a known
// credentials-bearing pattern. Match is case-insensitive.
func IsSecretFile(basename string) bool {
	lower := strings.ToLower(basename)
	if _, ok := secretFiles[lower]; ok {
		return true
	}
	for _, prefix := range secretFilePrefixes {
		if lower == prefix || strings.HasPrefix(lower, prefix+".") {
			return true
		}
	}
	return false
}

// IsSecretDir reports whether the given directory basename should be
// skipped wholesale because it typically contains credentials.
func IsSecretDir(basename string) bool {
	_, ok := secretDirs[strings.ToLower(basename)]
	return ok
}