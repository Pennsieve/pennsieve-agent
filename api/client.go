package api

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/config"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-go"
	"github.com/spf13/viper"
	"log"
)

func GetActiveUser() (*models.UserInfo, error) {
	// Get Active User or/and Authenticate
	_, err := config.InitializeDB()
	if err != nil {
		log.Println("Driver creation failed", err.Error())
		return nil, err
	}

	userSettings, _ := models.GetAllUserSettings()

	// If no entries are found in database, go with default profile in config
	if len(userSettings) <= 0 {
		fmt.Println("No record found in User Settings --> Creating new entry")

		selectedProfile := viper.GetString("global.default_profile")
		fmt.Println("Selected Profile: ", selectedProfile)

		apiToken := viper.GetString(selectedProfile + ".api_token")
		apiSecret := viper.GetString(selectedProfile + ".api_secret")

		viper.SetDefault(selectedProfile+".environment", "prod")
		environment := viper.GetString(selectedProfile + ".environment")

		client := pennsieve.NewClient()
		client.Authentication.Authenticate(apiToken, apiSecret)

		user, _ := client.User.GetUser(nil, nil)
		fmt.Println(user)

		if client.Credentials.IsRefreshed {
			client.Credentials.IsRefreshed = false
		}

		// Get Organization Information
		org, err := client.Organization.Get(nil, client.OrganizationNodeId)

		userParams := models.UserInfoParams{
			Id:               user.ID,
			Name:             user.Email,
			SessionToken:     client.Credentials.Token,
			RefreshToken:     client.Credentials.RefreshToken,
			Profile:          selectedProfile,
			Environment:      environment,
			OrganizationId:   client.OrganizationNodeId,
			OrganizationName: org.Organization.Name,
		}

		userInfo, err := models.CreateNewUserInfo(userParams)
		if err != nil {
			log.Println("NewUserInfo failed", err.Error())
			return nil, err
		}

		// Create userSettings
		userSettingsParams := models.UserSettingsParams{
			UserId:  userInfo.InnerId,
			Profile: selectedProfile,
		}
		models.CreateNewUserSettings(userSettingsParams)

		return userInfo, nil

	}

	// If entries found in database, continue with active profile
	fmt.Println("Found entry")

	// Return first entry. There should always be only 1 or 0 entries for the activeUser.
	currentUser := userSettings[0]

	currentUserInfo, _ := models.GetUserInfo(currentUser.UserId)

	return currentUserInfo, nil
}
