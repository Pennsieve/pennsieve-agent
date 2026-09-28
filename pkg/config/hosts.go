package config

import (
	"net/url"
	"os"
	"strings"

	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/spf13/viper"
)

// legacyAPI2Host is what every custom api_host used to get for its v2 API,
// whatever the api_host was. Kept only as the fallback for a custom host
// that is not an api.<domain> name (e.g. a localhost gateway).
const legacyAPI2Host = "https://api2.pennsieve.net"

// API2Host is the v2 (serverless) API that goes with apiHost.
//
// explicit (a profile's api2_host, or PENNSIEVE_API2_HOST) wins. Otherwise an
// api.<domain> host maps to api2.<domain>, so a profile pointed at
// https://api.pennsieve.ai reaches https://api2.pennsieve.ai rather than the
// dev API. An empty apiHost means the SDK defaults (production).
func API2Host(apiHost, explicit string) string {
	if e := strings.TrimRight(strings.TrimSpace(explicit), "/"); e != "" {
		return e
	}
	if strings.TrimSpace(apiHost) == "" {
		return pennsieve.BaseURLV2
	}
	u, err := url.Parse(strings.TrimSpace(apiHost))
	if err != nil || u.Host == "" {
		return legacyAPI2Host
	}
	if strings.HasPrefix(u.Host, "api.") {
		return u.Scheme + "://api2." + strings.TrimPrefix(u.Host, "api.")
	}
	return legacyAPI2Host
}

// ProfileAPIHosts returns (api, api2) for a config-file profile: its api_host
// and api2_host keys, or the SDK defaults when api_host is unset.
func ProfileAPIHosts(profile string) (string, string) {
	apiHost := viper.GetString(profile + ".api_host")
	if apiHost == "" {
		return pennsieve.BaseURLV1, API2Host("", viper.GetString(profile+".api2_host"))
	}
	return apiHost, API2Host(apiHost, viper.GetString(profile+".api2_host"))
}

// EnvAPIHosts returns (api, api2) from PENNSIEVE_API_HOST and
// PENNSIEVE_API2_HOST, or the SDK defaults when PENNSIEVE_API_HOST is unset.
func EnvAPIHosts() (string, string) {
	apiHost, present := os.LookupEnv("PENNSIEVE_API_HOST")
	if !present {
		return pennsieve.BaseURLV1, API2Host("", os.Getenv("PENNSIEVE_API2_HOST"))
	}
	return apiHost, API2Host(apiHost, os.Getenv("PENNSIEVE_API2_HOST"))
}
