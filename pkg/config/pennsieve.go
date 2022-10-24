// Package api Package contains method implementations that can be shared between CMDs
// in the CLI and which are not implemented in the gRPC server.

package config

import (
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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

	// useProfile is true when config.ini file exists and using credentials associated with profile.
	// It is false when we are using environment variables.
	useProfile := false

	// Set viper profile
	if viper.IsSet(userSettings.Profile + ".api_token") {
		viper.Set("pennsieve.api_token", viper.Get(userSettings.Profile+".api_token"))
		viper.Set("pennsieve.api_secret", viper.Get(userSettings.Profile+".api_secret"))
		useProfile = true
	}

	// Update baseURL if config specifies a custom API-HOST (such as https://api.pennsieve.net)
	if viper.IsSet(userSettings.Profile + ".api_host") {
		viper.Set("pennsieve.api_host", viper.GetString(userSettings.Profile+".api_host"))
		viper.Set("pennsieve.api_host2", "https://api2.pennsieve.net")
	} else {
		viper.Set("pennsieve.api_host2", pennsieve.BaseURLV2)
	}

	client := pennsieve.NewClient(
		viper.GetString("pennsieve.api_host"),
		viper.GetString("pennsieve.api_host2"))

	// Set the upload bucket to a custom specified bucket if specified in the config file.
	if viper.IsSet(userSettings.Profile + ".upload_bucket") {
		viper.Set("pennsieve.uploadBucket", viper.GetString(userSettings.Profile+".upload_bucket"))
	}
	if viper.IsSet("pennsieve.upload_bucket") {
		client.UploadBucket = viper.GetString("pennsieve.uploadBucket")
	}

	// Check if existing session token is expired.
	// Check Expiration Time for current session and refresh if necessary
	info, err := uiStore.GetUserInfo(userSettings.UserId, userSettings.Profile)
	if err != nil {
		log.Error(err)
	}

	currentApiToken := viper.GetString("pennsieve.api_token")
	currentApiSecret := viper.GetString("pennsieve.api_secret")

	client.APICredentials = pennsieve.APICredentials{
		ApiKey:    currentApiToken,
		ApiSecret: currentApiSecret,
	}

	// GET CREDENTIALS ASSOCIATED WITH CLIENT
	if time.Now().After(info.TokenExpire.Add(-5 * time.Minute)) {
		// Need to get new session token

		log.Info("Refreshing Pennsieve session token")

		session, err := client.Authentication.Authenticate(currentApiToken, currentApiSecret)

		if err != nil {
			log.Error("Error authenticating:", err)
			return nil, err
		}
		client.APISession = *session

		// Only store sessiontoken in the DB if we are using a profile.
		if useProfile {
			uiStore.UpdateTokenForUser(info, session)
		}

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
