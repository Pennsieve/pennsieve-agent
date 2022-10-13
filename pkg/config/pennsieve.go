// Package api Package contains method implementations that can be shared between CMDs
// in the CLI and which are not implemented in the gRPC server.

package config

import (
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/spf13/viper"
	"log"
	"time"
)

//var PennsieveClient *pennsieve.Client

// InitPennsieveClient initializes the Pennsieve Client.
func InitPennsieveClient(usStore store.UserSettingsStore, uiStore store.UserInfoStore) (*pennsieve.Client, error) {
	// Get current user-settings. This is either 0, or 1 entry.
	userSettings, err := usStore.Get()
	if err != nil {
		log.Fatalln("Could not get User Settings.")
	}

	apiV1Url := pennsieve.BaseURLV1
	apiV2Url := pennsieve.BaseURLV2

	// Update baseURL if config specifies a custom API-HOST (such as https://api.pennsieve.net)
	customAPIHost := viper.GetString(userSettings.Profile + ".api_host")
	if customAPIHost != "" {
		apiV1Url = customAPIHost
		apiV2Url = "https://api2.pennsieve.net"
	}

	client := pennsieve.NewClient(apiV1Url, apiV2Url)

	// Set the upload bucket to a custom specified bucket if specified in the config file.
	customUploadBucket := viper.GetString(userSettings.Profile + ".upload_bucket")
	if customUploadBucket != "" {
		client.UploadBucket = customUploadBucket
	}

	// Check if existing session token is expired.
	// Check Expiration Time for current session and refresh if necessary
	info, err := uiStore.GetUserInfo(userSettings.UserId, userSettings.Profile)
	if err != nil {
		log.Println(err)
	}

	apiToken := viper.GetString(userSettings.Profile + ".api_token")
	apiSecret := viper.GetString(userSettings.Profile + ".api_secret")

	client.APICredentials = pennsieve.APICredentials{
		ApiKey:    apiToken,
		ApiSecret: apiSecret,
	}

	// GET CREDENTIALS ASSOCIATED WITH CLIENT
	if time.Now().After(info.TokenExpire.Add(-5 * time.Minute)) {
		// Need to get new session token

		log.Println("Refreshing token", apiToken, apiSecret)

		// We are using reAuthenticate instead of refresh pathway as eventually, the refresh-token
		// also expires and there is no real reason why we don't just re-authenticate.`
		session, err := client.Authentication.Authenticate(apiToken, apiSecret)

		if err != nil {
			log.Println("Error authenticating:", err)
			return nil, err
		}
		client.APISession = *session

		uiStore.UpdateTokenForUser(info, session)
	} else {
		// Existing info has active token that can be used.

		client.APISession = pennsieve.APISession{
			Token:        info.SessionToken,
			IdToken:      info.IdToken,
			Expiration:   info.TokenExpire,
			RefreshToken: info.RefreshToken,
			IsRefreshed:  false,
		}

	}

	return client, err
}
