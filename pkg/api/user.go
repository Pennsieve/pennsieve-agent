// Package api Package contains method implementations that can be shared between CMDs
// in the CLI and which are not implemented in the gRPC server.

package api

import (
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
)

//GetActiveUserId returns userId of the active user
func GetActiveUserId() (string, error) {
	var clientSession models.UserSettings
	userSettings, err := clientSession.Get()
	if err != nil {
		return "", fmt.Errorf("No active user found in %s.\n",
			viper.ConfigFileUsed())
	}

	return userSettings.UserId, nil

}

// GetActiveUser returns the user that is currently set in UserSettings table
// This method does not update the active session, or handles situations where
// the active user is not set.
//
// Use the UpdateActiveUser method to handle those situations.
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

			// Create new user settings
			params := models.UserSettingsParams{
				UserId:  "",
				Profile: selectedProfile,
			}
			_, err = models.CreateNewUserSettings(params)
			if err != nil {
				fmt.Println("Error Creating new UserSettings")
				return nil, err
			}

			fmt.Printf("about to switch")

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
		return nil, err
	}

	return currentUserInfo, nil
}

// UpdateActiveUser returns userInfo for active user and updates local SQlite DB
func UpdateActiveUser() (*models.UserInfo, error) {

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

			// Create new user settings
			params := models.UserSettingsParams{
				UserId:  "",
				Profile: selectedProfile,
			}
			_, err = models.CreateNewUserSettings(params)
			if err != nil {
				fmt.Println("Error Creating new UserSettings")
				return nil, err
			}

			fmt.Printf("about to switch")

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

	PennsieveClient.APISession = pennsieve.APISession{
		Token:        currentUserInfo.SessionToken,
		IdToken:      currentUserInfo.IdToken,
		Expiration:   currentUserInfo.TokenExpire,
		RefreshToken: currentUserInfo.RefreshToken,
		IsRefreshed:  false,
	}

	apiToken := viper.GetString(userSettings.Profile + ".api_token")
	apiSecret := viper.GetString(userSettings.Profile + ".api_secret")

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

	// Directly update baseURL, so we can authenticate against new profile before setting up new Client
	customAPIHost := viper.GetString(profile + ".api_host")
	if customAPIHost != "" {
		PennsieveClient.SetBasePathForServices(customAPIHost, "https://api2.pennsieve.net")
	} else {
		PennsieveClient.SetBasePathForServices(pennsieve.BaseURLV1, pennsieve.BaseURLV2)
	}

	// Check credentials of new profile
	credentials, err := PennsieveClient.Authentication.Authenticate(apiToken, apiSecret)
	if err != nil {
		fmt.Println("Problem with authentication")
		return nil, err
	}

	// Get the User for the new profile
	existingUser, err := PennsieveClient.User.GetUser(nil, nil)
	if err != nil {
		fmt.Println("Problem with getting user", err)
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

	// Get UserInfo associated with settings or create if not exist.
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

func ReAuthenticate() (pennsieve.APISession, error) {
	apiSession, err := PennsieveClient.Authentication.ReAuthenticate()
	newSession := pennsieve.APISession{
		Token:        apiSession.Token,
		IdToken:      apiSession.IdToken,
		Expiration:   apiSession.Expiration,
		RefreshToken: apiSession.RefreshToken,
		IsRefreshed:  apiSession.IsRefreshed,
	}

	return newSession, err
}
