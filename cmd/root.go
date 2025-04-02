// Package cmd
// Copyright © 2022 University of Pennsylvania <support@server>>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cmd

import (
	"embed"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/cmd/download"
	"github.com/pennsieve/pennsieve-agent/cmd/map"
	"github.com/pennsieve/pennsieve-agent/cmd/timeseries"
	"log"
	"os"
	"path/filepath"

	"github.com/pennsieve/pennsieve-agent/cmd/account"
	"github.com/pennsieve/pennsieve-agent/cmd/agent"
	"github.com/pennsieve/pennsieve-agent/cmd/config"
	"github.com/pennsieve/pennsieve-agent/cmd/dataset"
	"github.com/pennsieve/pennsieve-agent/cmd/manifest"
	"github.com/pennsieve/pennsieve-agent/cmd/profile"
	"github.com/pennsieve/pennsieve-agent/cmd/upload"
	"github.com/pennsieve/pennsieve-agent/cmd/version"
	"github.com/pennsieve/pennsieve-agent/cmd/whoami"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var migrationsFS embed.FS

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pennsieve",
	Short: "A Command Line Interface for the Pennsieve Platform.",
	Long:  ``,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		// Initialize Viper before each command/subcommand
		// Except when user runs the setup config wizard
		if cmd.CommandPath() == "pennsieve config wizard" ||
			cmd.CommandPath() == "pennsieve config init" {
			return nil
		}

		return initViper()

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(fs embed.FS) {
	migrationsFS = fs
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(whoami.WhoamiCmd)
	rootCmd.AddCommand(config.ConfigCmd)
	rootCmd.AddCommand(profile.ProfileCmd)
	rootCmd.AddCommand(upload.UploadCmd)
	rootCmd.AddCommand(agent.AgentCmd)
	rootCmd.AddCommand(manifest.ManifestCmd)
	rootCmd.AddCommand(dataset.DatasetCmd)
	rootCmd.AddCommand(version.VersionCmd)
	rootCmd.AddCommand(account.AccountCmd)
	rootCmd.AddCommand(download.DownloadCmd)
	rootCmd.AddCommand(_map.MapCmd)
	rootCmd.AddCommand(timeseries.TimeseriesCmd)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.pennsieve/config.ini)")

}

// initConfig reads in config file and ENV variables if set.
func initViper() error {

	// initialize client after initializing Viper as it needs viper to get api key/secret
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".pennsieve" (without extension).
		viper.SetConfigType("ini")
		viper.AddConfigPath(filepath.Join(home, ".pennsieve"))

	}

	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".pennsieve/pennsieve_agent.db")

	migrationsPath := filepath.Join(home, ".pennsieve", "migrations")
	err := os.MkdirAll(migrationsPath, os.ModePerm)
	if err != nil {
		log.Fatal("Error creating temp dir:", err)
	}
	//migrationPath := extractMigrations(migrationsPath)

	viper.SetDefault("global.default_profile", "pennsieve")
	viper.SetDefault("agent.db_path", dbPath)
	viper.SetDefault("agent.useConfigFile", true)
	// Internal agent filepath
	viper.SetDefault("migration.path", fmt.Sprintf(filepath.Join("file:", "//", migrationsPath)))

	// Filepath on user system
	viper.SetDefault("migration.local", fmt.Sprintf(migrationsPath))

	log.Println("root")
	log.Println("root.go migration.path:", viper.GetString("migration.path"))
	log.Println("root.go migration.local:", viper.GetString("migration.local"))

	err = extractMigrations(migrationsFS, migrationsPath)

	workers := os.Getenv("PENNSIEVE_AGENT_UPLOAD_WORKERS")
	if len(workers) > 0 {
		viper.Set("agent.upload_workers", os.Getenv("PENNSIEVE_AGENT_UPLOAD_WORKERS"))
	} else {
		viper.SetDefault("agent.upload_workers", "10") // Number of concurrent files during upload
	}

	port := os.Getenv("PENNSIEVE_AGENT_PORT")
	if len(port) > 0 {
		viper.Set("agent.port", os.Getenv("PENNSIEVE_AGENT_PORT"))
	} else {
		viper.SetDefault("agent.port", "9000")
	}

	chunkSize := os.Getenv("PENNSIEVE_AGENT_CHUNK_SIZE")
	if len(chunkSize) > 0 {
		viper.Set("agent.upload_chunk_size", os.Getenv("PENNSIEVE_AGENT_CHUNK_SIZE"))
	} else {
		viper.SetDefault("agent.upload_chunk_size", "32")
	}

	apiKey := os.Getenv("PENNSIEVE_API_KEY")
	// use API Key and TOKEN from ENV vars if they exist
	if len(apiKey) > 0 {
		viper.Set("pennsieve.api_token", apiKey)

		apiSecret := os.Getenv("PENNSIEVE_API_SECRET")
		if len(apiSecret) == 0 {
			fmt.Println("Need to set PENNSIEVE_API_SECRET when PENNSIEVE_API_KEY is set as an ENV variable")
			os.Exit(1)
		}
		viper.Set("pennsieve.api_secret", apiSecret)
		viper.Set("agent.useConfigFile", false)

	} else {
		// Load from config file if it exists
		if err := viper.ReadInConfig(); err != nil {
			if viper.GetBool("agent.useConfigFile") {
				fmt.Println("No Pennsieve configuration file exists.")
				fmt.Println("\nPlease use `pennsieve config wizard` to setup your Pennsieve profile, or")
				fmt.Println("\nset the PENNSIEVE_API_KEY and PENNSIEVE_API_SECRET environment variables.")
				os.Exit(1)
			}
		}
	}

	return nil
}

func extractMigrations(fs embed.FS, targetDir string) error {
	files, err := fs.ReadDir("db/migrations")
	if err != nil {
		return fmt.Errorf("failed to read embedded migration files: %w", err)
	}

	for _, file := range files {
		filePath := filepath.Join(targetDir, file.Name())

		// Read the embedded file content
		data, err := fs.ReadFile("db/migrations/" + file.Name())
		if err != nil {
			return fmt.Errorf("failed to read embedded migration file %s: %w", file.Name(), err)
		}

		// Write the file to the target directory
		err = os.WriteFile(filePath, data, 0644)
		if err != nil {
			return fmt.Errorf("failed to write migration file %s: %w", filePath, err)
		}
	}

	return nil
}
