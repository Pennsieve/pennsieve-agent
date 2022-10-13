// Package config contains method implementations related to the local database that can be shared between CMDs
// in the CLI and which are not implemented in the gRPC server.

package config

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pennsieve/pennsieve-agent/migrations"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

// InitializeDB initialized local SQL DB and creates userinfo for current user.
// This method returns a sql.config instance and:
// 1. Ensures that this config has the correct tables
// 2. Ensures that the userSettings table has a single valid entry
// 3. Ensures that the userInfo table has a valid entry
func InitializeDB() (*sql.DB, error) {
	// Initialize connection to the database
	var err error
	home, err := os.UserHomeDir()
	dbPath := filepath.Join(home, ".pennsieve/pennsieve_agent.db")
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&mode=rwc&_journal_mode=WAL")

	userSettingsStore := store.NewUserSettingsStore(db)
	userInfoStore := store.NewUserInfoStore(db)

	// Get current user-settings. This is either 0, or 1 entry.
	_, err = userSettingsStore.Get()
	if err != nil {
		if err == sql.ErrNoRows || strings.ContainsAny(err.Error(), "no such table") {
			// The database does not exist or no userSettings are defined in the table.

			// If the table does not exist, run migrations.
			if strings.ContainsAny(err.Error(), "no such table") {
				log.Info("Setting up the local database and running migrations.")
				migrations.Run(db)
			}

			// Select the globally defined profile.
			selectedProfile := viper.GetString("global.default_profile")
			log.Info("Using default profile name: ", selectedProfile)

			if selectedProfile == "" {
				log.Fatalf("No default profile defined in %s. Please update configuration.\n\n",
					viper.ConfigFileUsed())
			}

			// Check if profile exist
			isSet := viper.IsSet(selectedProfile + ".api_token")
			if !isSet {
				return nil, errors.New(fmt.Sprintf("Profile not found: %s", selectedProfile))
			}

			// Profile exists, verify login and refresh token if necessary
			apiToken := viper.GetString(selectedProfile + ".api_token")
			apiSecret := viper.GetString(selectedProfile + ".api_secret")
			environment := viper.GetString(selectedProfile + ".env")
			customUploadBucket := viper.GetString(selectedProfile + ".upload_bucket")

			client := pennsieve.NewClient(pennsieve.BaseURLV1, pennsieve.BaseURLV2)

			if customUploadBucket != "" {
				client.UploadBucket = customUploadBucket
			}

			// Directly update baseURL, so we can authenticate against new profile before setting up new Client
			customAPIHost := viper.GetString(selectedProfile + ".api_host")
			if customAPIHost != "" {
				client.SetBasePathForServices(customAPIHost, "https://api2.pennsieve.net")
			} else {
				client.SetBasePathForServices(pennsieve.BaseURLV1, pennsieve.BaseURLV2)
			}

			// Check credentials of new profile
			credentials, err := client.Authentication.Authenticate(apiToken, apiSecret)
			if err != nil {
				log.Error("Problem with authentication")
				return nil, err
			}

			// Get the User for the new profile
			existingUser, err := client.User.GetUser(nil, nil)
			if err != nil {
				log.Error("Problem with getting user", err)
				return nil, err
			}

			org, err := client.Organization.Get(nil, client.OrganizationNodeId)
			if err != nil {
				log.Error("Error getting organization")
				return nil, err
			}

			infoParams := store.UserInfoParams{
				Id:               existingUser.ID,
				Name:             existingUser.FirstName + " " + existingUser.LastName,
				SessionToken:     credentials.Token,
				RefreshToken:     credentials.RefreshToken,
				Profile:          selectedProfile,
				Environment:      environment,
				OrganizationId:   client.OrganizationNodeId,
				OrganizationName: org.Organization.Name,
			}
			_, err = userInfoStore.CreateNewUserInfo(infoParams)
			if err != nil {
				log.Fatalln(err)
			}

			// Create new user settings
			params := store.UserSettingsParams{
				UserId:  existingUser.ID,
				Profile: selectedProfile,
			}
			_, err = userSettingsStore.CreateNewUserSettings(params)
			if err != nil {
				log.Fatalln("Error Creating new UserSettings")
			}
		} else {
			log.Fatalln(err)
		}

	}

	return db, err
}
