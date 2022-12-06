// Package api Package contains method implementations that can be shared between CMDs
// in the CLI and which are not implemented in the gRPC server.

package config

import (
	"context"
	"errors"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"time"
)

// InitPennsieveClient initializes the Pennsieve Client.
func InitPennsieveClient(usStore store.UserSettingsStore, uiStore store.UserInfoStore) (*pennsieve.Client, error) {

	activeConfig := pennsieve.APIParams{
		Port:          viper.GetString("agent.port"),
		UploadBucket:  pennsieve.DefaultUploadBucket,
		UseConfigFile: true,
	}

	var client *pennsieve.Client

	if viper.GetBool("agent.useConfigFile") {
		// USE CONFIG INI AND PROFILE

		// Get current user-settings. This is either 0, or 1 entry.
		userSettings, err := usStore.Get()

		if userSettings == nil || userSettings.Profile == "N/A" {
			usStore.Delete()
			userSettings = &store.UserSettings{
				UserId:          "",
				Profile:         viper.GetString("global.default_profile"),
				UseDatasetId:    "",
				UploadSessionId: "",
			}
		}

		// If there is not userSetting, or the profile that was set in the userSettings table no longer exists
		// in the config file, create a new settings
		if err != nil || !viper.IsSet(userSettings.Profile+".api_token") {
			userSettings = &store.UserSettings{
				UserId:          "",
				Profile:         viper.GetString("global.default_profile"),
				UseDatasetId:    "",
				UploadSessionId: "",
			}
		}

		activeConfig.ApiKey = viper.GetString(userSettings.Profile + ".api_token")
		activeConfig.ApiSecret = viper.GetString(userSettings.Profile + ".api_secret")
		if activeConfig.ApiKey == "" || activeConfig.ApiSecret == "" {
			return nil, errors.New(fmt.Sprintf("API Token/secret not set for profile: %s", userSettings.Profile))
		}

		// Update baseURL if config specifies a custom API-HOST (such as https://api.pennsieve.net)
		if viper.IsSet(userSettings.Profile + ".api_host") {
			activeConfig.ApiHost = viper.GetString(userSettings.Profile + ".api_host")
			activeConfig.ApiHost2 = "https://api2.pennsieve.net"
		} else {
			activeConfig.ApiHost = pennsieve.BaseURLV1
			activeConfig.ApiHost2 = pennsieve.BaseURLV2
		}

		if viper.IsSet(userSettings.Profile + ".upload_bucket") {
			activeConfig.UploadBucket = viper.GetString(userSettings.Profile + ".upload_bucket")
		} else {
			activeConfig.UploadBucket = pennsieve.DefaultUploadBucket
		}

		client = pennsieve.NewClient(activeConfig)

		// Check if existing session token is expired.
		// Check Expiration Time for current session and refresh if necessary
		info, err := uiStore.GetUserInfo(userSettings.UserId, userSettings.Profile)
		if err != nil {
			fmt.Println("CREATE SETTINGS AND INFO")

			// Check credentials of new profile
			credentials, err := client.Authentication.Authenticate(activeConfig.ApiKey, activeConfig.ApiSecret)
			if err != nil {
				log.Error("Problem with authentication", err)
				return nil, err
			}

			client.APISession = pennsieve.APISession{
				Token:        credentials.Token,
				IdToken:      credentials.IdToken,
				Expiration:   credentials.Expiration,
				RefreshToken: credentials.RefreshToken,
				IsRefreshed:  false,
			}

			// Get the User for the new profile
			existingUser, err := client.User.GetUser(nil)
			if err != nil {
				log.Error("Problem with getting user", err)
				return nil, err
			}

			currentOrg, err := client.Organization.Get(context.Background(), existingUser.PreferredOrganization)

			params := store.UserSettingsParams{
				UserId:  existingUser.ID,
				Profile: userSettings.Profile,
			}

			_, err = usStore.CreateNewUserSettings(params)
			if err != nil {
				fmt.Println("Error Creating new UserSettings")
				return nil, err
			}

			uiParams := store.UserInfoParams{
				Id:               existingUser.ID,
				Name:             existingUser.FirstName + " " + existingUser.LastName,
				SessionToken:     credentials.Token,
				RefreshToken:     credentials.RefreshToken,
				TokenExpire:      credentials.Expiration,
				Profile:          userSettings.Profile,
				IdToken:          "",
				Environment:      "",
				OrganizationId:   existingUser.PreferredOrganization,
				OrganizationName: currentOrg.Organization.Name,
			}

			client.OrganizationNodeId = currentOrg.Organization.ID

			info, err = uiStore.CreateNewUserInfo(uiParams)
			if err != nil {
				log.Error(err)
			}

		}

		if time.Now().After(info.TokenExpire.Add(-5 * time.Minute)) {
			// Need to get new session token

			log.Info("Refreshing Pennsieve session token")

			session, err := client.Authentication.Authenticate(activeConfig.ApiKey, activeConfig.ApiSecret)

			if err != nil {
				log.Error("Error authenticating:", err)
				return nil, err
			}
			client.APISession = *session

			err = uiStore.UpdateTokenForUser(info.Id, session)
			if err != nil {
				return nil, err
			}

			info.SessionToken = session.Token
			info.RefreshToken = session.RefreshToken
			info.TokenExpire = session.Expiration
			info.IdToken = session.IdToken

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

	} else {
		// USE ENVIRONMENT VARIABLES
		fmt.Println("USE ENVIRONMENT VARIABLES")

		activeConfig.UseConfigFile = false
		activeConfig.Profile = ""
		activeConfig.ApiKey = os.Getenv("PENNSIEVE_API_KEY")
		activeConfig.ApiSecret = os.Getenv("PENNSIEVE_API_SECRET")
		uploadBucket, present := os.LookupEnv("PENNSIEVE_UPLOAD_BUCKET")
		if present {
			activeConfig.UploadBucket = uploadBucket
		}

		apiHost, present := os.LookupEnv("PENNSIEVE_API_HOST")
		if present {
			activeConfig.ApiHost = apiHost
			activeConfig.ApiHost2 = "https://api2.pennsieve.net"
		} else {
			activeConfig.ApiHost = pennsieve.BaseURLV1
			activeConfig.ApiHost2 = pennsieve.BaseURLV2
		}

		client = pennsieve.NewClient(activeConfig)

		session, err := client.Authentication.Authenticate(activeConfig.ApiKey, activeConfig.ApiSecret)
		if err != nil {
			log.Error("Error authenticating:", err)
			return nil, err
		}
		client.APISession = *session

		// Get the User for the new profile
		existingUser, err := client.User.GetUser(nil)
		if err != nil {
			log.Error("Problem with getting user", err)
			return nil, err
		}

		currentOrg, err := client.Organization.Get(context.Background(), existingUser.PreferredOrganization)

		params := store.UserSettingsParams{
			UserId:  existingUser.ID,
			Profile: "N/A",
		}

		err = usStore.Delete()
		if err != nil {
			return nil, err
		}

		_, err = usStore.CreateNewUserSettings(params)
		if err != nil {
			fmt.Println("Error Creating new UserSettings")
			return nil, err
		}

		// Delete existing ENV Variable User_Info
		err = uiStore.DeleteUserForProfile("N/A")
		if err != nil {
			return nil, err
		}

		uiParams := store.UserInfoParams{
			Id:               existingUser.ID,
			Name:             existingUser.FirstName + " " + existingUser.LastName,
			SessionToken:     session.Token,
			RefreshToken:     session.RefreshToken,
			TokenExpire:      session.Expiration,
			Profile:          "N/A",
			IdToken:          "",
			Environment:      "",
			OrganizationId:   existingUser.PreferredOrganization,
			OrganizationName: currentOrg.Organization.Name,
		}

		client.OrganizationNodeId = currentOrg.Organization.ID

		_, err = uiStore.CreateNewUserInfo(uiParams)
		if err != nil {
			log.Error(err)
		}

	}

	return client, nil
}
