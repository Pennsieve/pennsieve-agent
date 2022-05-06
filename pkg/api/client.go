// Package api provides methods that leverage the local SQLite DB ang the Pennsieve Client.
package api

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/pennsieve/pennsieve-go"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
	"path/filepath"
)

// GetActiveUser returns userInfo for active user and updates local SQlite DB
func GetActiveUser() (*models.UserInfo, error) {

	// Get current user-settings. This is either 0, or 1 entry.
	var clientSession models.UserSettings
	userSettings, err := clientSession.Get()

	if err != nil {
		// If no entry is found in database, check default profile in db and setup DB
		if errors.Is(err, &models.NoClientSessionError{}) {
			fmt.Println("No record found in User Settings --> Checking Default Profile.")

			selectedProfile := viper.GetString("global.default_profile")
			fmt.Println("Selected Profile: ", selectedProfile)

			if selectedProfile == "" {
				return nil, fmt.Errorf("No default profile defined in %s. Please update configuration.\n",
					viper.ConfigFileUsed())
			}

			apiToken := viper.GetString(selectedProfile + ".api_token")
			apiSecret := viper.GetString(selectedProfile + ".api_secret")

			_, err := PennsieveClient.Authentication.Authenticate(apiToken, apiSecret)
			if err != nil {
				return nil, err
			}

			currentUser, err := SwitchUser(selectedProfile)
			if err != nil {
				fmt.Println("Error switching user.")
				return nil, err
			}

			return currentUser, nil
		} else {
			return nil, err
		}

	}

	// If entries found in database, continue with active profile
	currentUserInfo, err := models.GetUserInfo(userSettings.UserId, userSettings.Profile)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No userInfo found for user settings")
			_, err := SwitchUser(userSettings.Profile)
			if err != nil {
				fmt.Println("error switching user:", err)
			}

		} else {
			log.Fatal(err)
		}

		return nil, err
	}

	// Update baseURL if db specifies a custom API-HOST (such as https://api.pennsieve.net)
	customAPIHost := viper.GetString(userSettings.Profile + ".api_host")
	if customAPIHost != "" {
		//fmt.Println("Using custom API-Host: ", customAPIHost)
		PennsieveClient.BaseURL = customAPIHost
	}

	apiToken := viper.GetString(userSettings.Profile + ".api_token")
	apiSecret := viper.GetString(userSettings.Profile + ".api_secret")

	PennsieveClient.APISession = pennsieve.APISession{
		Token:        currentUserInfo.SessionToken,
		IdToken:      currentUserInfo.IdToken,
		Expiration:   currentUserInfo.TokenExpire,
		RefreshToken: currentUserInfo.RefreshToken,
		IsRefreshed:  false,
	}

	PennsieveClient.APICredentials = pennsieve.APICredentials{
		ApiKey:    apiToken,
		ApiSecret: apiSecret,
	}

	return currentUserInfo, nil
}

// SwitchUser switches between profiles and returns active userInfo.
func SwitchUser(profile string) (*models.UserInfo, error) {
	// Check if profile exist
	isSet := viper.IsSet(profile + ".api_token")
	if !isSet {
		fmt.Printf("Profile %s not found\n", profile)
		return nil, fmt.Errorf("")
	}

	// Profile exists, verify login and refresh token if necessary
	apiToken := viper.GetString(profile + ".api_token")
	apiSecret := viper.GetString(profile + ".api_secret")
	environment := viper.GetString(profile + ".env")

	// Update baseURL if db specifies a custom API-HOST (such as https://api.pennsieve.net)
	customAPIHost := viper.GetString(profile + ".api_host")
	if customAPIHost != "" {
		//fmt.Println("Using custom API-Host: ", customAPIHost)
		PennsieveClient.BaseURL = customAPIHost
	}

	credentials, err := PennsieveClient.Authentication.Authenticate(apiToken, apiSecret)
	if err != nil {
		fmt.Println("Problem with authentication")
		return nil, err
	}

	existingUser, err := PennsieveClient.User.GetUser(nil, nil)
	if err != nil {
		fmt.Println("Problem with getting user")
		return nil, err
	}

	// Drop existing user settings
	_, err = db.DB.Exec("DELETE FROM user_settings;")
	if err != nil {
		return nil, err
	}

	// Create new user settings
	params := models.UserSettingsParams{
		UserId:  existingUser.ID,
		Profile: profile,
	}
	_, err = models.CreateNewUserSettings(params)
	if err != nil {
		fmt.Println("Error Creating new UserSettings")
		return nil, err
	}

	// Get UserInfo associated with settings
	newUserInfo, err := models.GetUserInfo(existingUser.ID, profile)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No userInfo found --> Creating new userinfo")

			org, err := PennsieveClient.Organization.Get(nil, PennsieveClient.OrganizationNodeId)
			if err != nil {
				fmt.Println("Error getting organization")
				return nil, err
			}

			params := models.UserInfoParams{
				Id:               existingUser.ID,
				Name:             existingUser.FirstName + " " + existingUser.LastName,
				SessionToken:     credentials.Token,
				RefreshToken:     credentials.RefreshToken,
				Profile:          profile,
				Environment:      environment,
				OrganizationId:   PennsieveClient.OrganizationNodeId,
				OrganizationName: org.Organization.Name,
			}
			newUserInfo, err = models.CreateNewUserInfo(params)
			if err != nil {
				fmt.Println("Error creating new userinfo ")
				return nil, err
			}

		} else {
			log.Fatal(err)
		}
	}

	return newUserInfo, nil
}

func AddUploadRecords(paths []string, basePath string, sessionId string) error {

	var records []models.UploadRecordParams
	for _, row := range paths {
		newRecord := models.UploadRecordParams{
			SourcePath: row,
			TargetPath: filepath.Join(basePath, row),
			S3Key:      uuid.New().String(),
			SessionID:  sessionId,
		}
		records = append(records, newRecord)
	}

	var record models.UploadRecord
	err := record.Add(records)
	if err != nil {
		log.Println("Error with AddUploadRecords: ", err)
		return err
	}

	return nil
}
