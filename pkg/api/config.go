// Package api Package contains method implementations that can be shared between CMDs
// in the CLI and which are not implemented in the gRPC server.

package api

import (
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/spf13/viper"
	"log"
)

var PennsieveClient *pennsieve.Client
var ActiveUser *models.UserInfo

// InitializeAPI initializes the Pennsieve Client.
func InitializeAPI() error {
	// Initialize Pennsieve Client

	// Get current user-settings. This is either 0, or 1 entry.
	var clientSession models.UserSettings
	userSettings, err := clientSession.Get()
	if err != nil {
		log.Fatalln("Could not get User Settings.")
	}

	apiV1Url := pennsieve.BaseURLV1
	apiV2Url := pennsieve.BaseURLV2

	// Update baseURL if db specifies a custom API-HOST (such as https://api.pennsieve.net)
	customAPIHost := viper.GetString(userSettings.Profile + ".api_host")
	if customAPIHost != "" {
		apiV1Url = customAPIHost
		apiV2Url = "https://api2.pennsieve.net"

	}

	PennsieveClient = pennsieve.NewClient(apiV1Url, apiV2Url)

	// Set the upload bucket to a custom specified bucket if specified in the config file.
	customUploadBucket := viper.GetString(userSettings.Profile + ".upload_bucket")
	if customUploadBucket != "" {
		PennsieveClient.UploadBucket = customUploadBucket
	}

	return err
}
