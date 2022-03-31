// Package api provides methods that leverage the local SQLite DB ang the Pennsieve Client.
package api

import (
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/config"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-go"
	"github.com/spf13/viper"
	"log"
)

// PennsieveClient represents the client from the Pennsieve-Go library
var PennsieveClient *pennsieve.Client

// GetActiveUser returns userInfo for active user and updates local SQlite DB
func GetActiveUser() (*models.UserInfo, error) {

	// Get current user-settings. This is either 0, or 1 entry.
	userSettings, _ := models.GetAllUserSettings()

	// If no entry is found in database, check default profile in config and setup DB
	if len(userSettings) <= 0 {
		fmt.Println("No record found in User Settings --> Checking Default Profile.")

		selectedProfile := viper.GetString("global.default_profile")
		fmt.Println("Selected Profile: ", selectedProfile)

		if selectedProfile == "" {
			return nil, fmt.Errorf("No default profile defined in %s. Please update configuration.\n",
				viper.ConfigFileUsed())
		}

		apiToken := viper.GetString(selectedProfile + ".api_token")
		apiSecret := viper.GetString(selectedProfile + ".api_secret")

		client := pennsieve.NewClient()
		_, err := client.Authentication.Authenticate(apiToken, apiSecret)
		if err != nil {
			return nil, err
		}

		currentUser, err := SwitchUser(selectedProfile)
		if err != nil {
			fmt.Println("Error switching user.")
			return nil, err
		}

		return currentUser, nil

	}

	// If entries found in database, continue with active profile
	// Return first entry. There should always be only 1 or 0 entries for the activeUser.
	currentUser := userSettings[0]

	currentUserInfo, err := models.GetUserInfo(currentUser.UserId, currentUser.Profile)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No userInfo found for user settings")
			_, err := SwitchUser(currentUser.Profile)
			if err != nil {
				fmt.Println("error switching user:", err)
			}

		} else {
			log.Fatal(err)
		}

		return nil, err
	}

	return currentUserInfo, nil
}

// SwitchUser SwtichUser
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

	PennsieveClient = pennsieve.NewClient()
	client := *PennsieveClient
	credentials, err := client.Authentication.Authenticate(apiToken, apiSecret)
	if err != nil {
		fmt.Println("Problem with authentication")
		return nil, err
	}
	existingUser, err := PennsieveClient.User.GetUser(nil, nil)
	if err != nil {
		fmt.Println("Problem with getting user")
		return nil, err
	}

	// Update the UserSettings DB entry
	_, err = models.GetAllUserSettings()
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No user settings found --> creating new user settings")
		} else {
			log.Fatal(err)
			return nil, err
		}
	}

	// Drop existing user settings
	_, err = config.DB.Exec("DELETE FROM user_settings;")
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
