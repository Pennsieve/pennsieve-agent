package config

import (
	"testing"

	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/spf13/viper"
)

func TestAPI2HostFollowsTheAPIHostDomain(t *testing.T) {
	cases := map[string]string{
		"https://api.pennsieve.ai":  "https://api2.pennsieve.ai",
		"https://api.pennsieve.net": "https://api2.pennsieve.net", // dev: same as before
		"https://api.pennsieve.io/": "https://api2.pennsieve.io",
		"http://localhost:8080":     legacyAPI2Host, // not api.<domain>: old behaviour
		"":                          pennsieve.BaseURLV2,
	}
	for in, want := range cases {
		if got := API2Host(in, ""); got != want {
			t.Errorf("API2Host(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExplicitAPI2HostWins(t *testing.T) {
	if got := API2Host("https://api.pennsieve.ai", " https://api2.example.org/ "); got != "https://api2.example.org" {
		t.Errorf("got %q", got)
	}
}

func TestProfileAPIHosts(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	viper.Set("clin.api_host", "https://api.pennsieve.ai")
	viper.Set("custom.api_host", "https://api.pennsieve.ai")
	viper.Set("custom.api2_host", "https://api2.other.org")

	if a, a2 := ProfileAPIHosts("clin"); a != "https://api.pennsieve.ai" || a2 != "https://api2.pennsieve.ai" {
		t.Errorf("clin: %q %q", a, a2)
	}
	if _, a2 := ProfileAPIHosts("custom"); a2 != "https://api2.other.org" {
		t.Errorf("custom: %q", a2)
	}
	if a, a2 := ProfileAPIHosts("default"); a != pennsieve.BaseURLV1 || a2 != pennsieve.BaseURLV2 {
		t.Errorf("default: %q %q", a, a2)
	}
}

func TestEnvAPIHosts(t *testing.T) {
	t.Setenv("PENNSIEVE_API_HOST", "https://api.pennsieve.ai")
	t.Setenv("PENNSIEVE_API2_HOST", "")
	if a, a2 := EnvAPIHosts(); a != "https://api.pennsieve.ai" || a2 != "https://api2.pennsieve.ai" {
		t.Errorf("env: %q %q", a, a2)
	}
}
