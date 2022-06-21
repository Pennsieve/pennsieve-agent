/*
Copyright © 2022 University of Pennsylvania <support@server>>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/cmd/agent"
	"github.com/pennsieve/pennsieve-agent/cmd/config"
	"github.com/pennsieve/pennsieve-agent/cmd/dataset"
	"github.com/pennsieve/pennsieve-agent/cmd/manifest"
	"github.com/pennsieve/pennsieve-agent/cmd/profile"
	"github.com/pennsieve/pennsieve-agent/cmd/upload"
	"github.com/pennsieve/pennsieve-agent/cmd/whoami"
	"github.com/pennsieve/pennsieve-agent/migrations"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	dbConfig "github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pennsieve-server",
	Short: "A brief description of your application",
	Long:  ``,

	PersistentPostRun: func(cmd *cobra.Command, args []string) {

		/*
			if Pennsieve Client APISession is set --> check if token is updated
			Pennsieve credentials are set when the command uses the Pennsieve REST API.
			If this is the case, we should check if the Pennsieve Go Library re-authenticated
			due to an expired token and update the UserInfo object in the local database to
			cache the updated session-token so next calls do not require re-authentication.
		*/

		if api.PennsieveClient != nil {

			creds := api.PennsieveClient.APISession
			if creds != (pennsieve.APISession{}) && creds.IsRefreshed {
				fmt.Println("Client credentials updated --> Update session token in UserInfo")
				models.UpdateTokenForUser(api.ActiveUser, &api.PennsieveClient.APISession)
			}
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(whoami.WhoamiCmd)
	rootCmd.AddCommand(config.ConfigCmd)
	rootCmd.AddCommand(profile.ProfileCmd)
	rootCmd.AddCommand(upload.UploadCmd)
	rootCmd.AddCommand(agent.AgentCmd)
	rootCmd.AddCommand(manifest.ManifestCmd)
	rootCmd.AddCommand(dataset.DatasetCmd)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "db", "",
		"db file (default is $HOME/.pennsieve/db.ini)")

	rootCmd.Flags().BoolP("toggle", "t", false,
		"Help message for toggle")
}

// initConfig reads in db file and ENV variables if set.
func initConfig() {

	// initialize client after initializing Viper as it needs viper to get api key/secret
	if cfgFile != "" {
		// Use db file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search db in home directory with name ".pennsieve-server" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("ini")
		viper.AddConfigPath(filepath.Join(home, ".pennsieve"))

		// Set viper defaults
		viper.SetDefault("env", "prod")
		viper.SetDefault("agent.port", "9000")
		viper.SetDefault("agent.upload_workers", "5")     // Number of concurrent files during upload
		viper.SetDefault("agent.upload_chunk_size", "32") // Upload chunk-size in MB
		viper.SetDefault("api_host", "https://api.pennsieve.io")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a db file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading db file:", viper.ConfigFileUsed())
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

	api.InitializeAPI()

	api.GetActiveUser()
}
