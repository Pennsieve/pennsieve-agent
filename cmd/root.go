// Package cmd
//Copyright © 2022 University of Pennsylvania <support@server>>
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
package cmd

import (
	"fmt"
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
	"os"
	"path/filepath"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pennsieve",
	Short: "A Command Line Interface for the Pennsieve Platform.",
	Long:  ``,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		// Initialize Viper before each command/subcommand
		// Except when user runs the setup config wizard
		if cmd.CommandPath() == "pennsieve config wizard" {
			return nil
		}

		return initViper()

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
	rootCmd.AddCommand(whoami.WhoamiCmd)
	rootCmd.AddCommand(config.ConfigCmd)
	rootCmd.AddCommand(profile.ProfileCmd)
	rootCmd.AddCommand(upload.UploadCmd)
	rootCmd.AddCommand(agent.AgentCmd)
	rootCmd.AddCommand(manifest.ManifestCmd)
	rootCmd.AddCommand(dataset.DatasetCmd)
	rootCmd.AddCommand(version.VersionCmd)

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

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("No Pennsieve configuration file exists.")
		fmt.Println("\nPlease use `pennsieve config wizard` to setup your Pennsieve profile.")
		os.Exit(1)
	}

	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".pennsieve/pennsieve_agent.db")

	viper.SetDefault("agent.port", "9000")
	viper.SetDefault("agent.upload_workers", "10")    // Number of concurrent files during upload
	viper.SetDefault("agent.upload_chunk_size", "32") // Upload chunk-size in MB
	viper.SetDefault("global.default_profile", "user")
	viper.SetDefault("agent.db_path", dbPath)

	viper.AutomaticEnv() // read in environment variables that match

	return nil
}
