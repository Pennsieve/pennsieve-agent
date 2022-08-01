package config

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/migrations"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	dbConfig "github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/spf13/viper"
	"log"
	"os"
)

//InitDB is used in every (except config) CMD to initialize configuration and DB.
func InitDB() {

	// Read configuration variables from config.ini file.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("No Pennsieve configuration file exists.")
		fmt.Println("\nPlease use `pennsieve-agent config wizard` to setup your Pennsieve profile.")
		os.Exit(1)
	}

	// Initialize SQLITE database
	_, err := dbConfig.InitializeDB()

	// Get current user-settings. This is either 0, or 1 entry.
	var clientSession models.UserSettings
	_, err = clientSession.Get()
	if err != nil {
		fmt.Println("Setup database")
		migrations.Run()

		selectedProfile := viper.GetString("global.default_profile")
		fmt.Println("Selected Profile: ", selectedProfile)

		if selectedProfile == "" {
			log.Fatalf("No default profile defined in %s. Please update configuration.\n\n",
				viper.ConfigFileUsed())
		}

		// Create new user settings
		params := models.UserSettingsParams{
			UserId:  "",
			Profile: selectedProfile,
		}
		_, err = models.CreateNewUserSettings(params)
		if err != nil {
			log.Fatalln("Error Creating new UserSettings")
		}

	}

	err = api.InitializeAPI()
	if err != nil {
		log.Fatalln("Unable to initialize API: ", err)
	}

	_, err = api.UpdateActiveUser()
	if err != nil {
		log.Fatalln("Unable to get active user: ", err)
	}
}
