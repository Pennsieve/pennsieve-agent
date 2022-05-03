/*
Copyright © 2022 University of Pennsylvania <support@agent>>

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
	"github.com/pennsieve/pennsieve-agent/cmd/manifest"
	"github.com/pennsieve/pennsieve-agent/cmd/profile"
	"github.com/pennsieve/pennsieve-agent/cmd/upload"
	"github.com/pennsieve/pennsieve-agent/cmd/whoami"
	dbConfig "github.com/pennsieve/pennsieve-agent/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pennsieve-agent",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,

	// TODO: Bring back to prevent reauth on every CLI invokation
	//PersistentPostRun: func(cmd *cobra.Command, args []string) {
	//
	//	// if Pennsieve Client set --> check if token is updated
	//	client := pennsieve.NewClient()
	//	fmt.Println("Client specified --> Check API Token")
	//	user, _ := api.GetActiveUser(client)
	//	models.UpdateTokenForUser(*user, client.Credentials)
	//
	//},
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
	_, err := dbConfig.InitializeDB()
	if err != nil {
		log.Println("Driver creation failed", err.Error())
	}

	rootCmd.AddCommand(whoami.WhoamiCmd)
	rootCmd.AddCommand(config.ConfigCmd)
	rootCmd.AddCommand(profile.ProfileCmd)
	rootCmd.AddCommand(upload.UploadCmd)
	rootCmd.AddCommand(agent.AgentCmd)
	rootCmd.AddCommand(manifest.ManifestCmd)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.pennsieve/config.ini)")

	rootCmd.Flags().BoolP("toggle", "t", false,
		"Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".pennsieve-agent" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("ini")
		viper.AddConfigPath(filepath.Join(home, ".pennsieve"))

		// Set viper defaults
		viper.SetDefault("env", "prod")
		viper.SetDefault("agent_port", "9000")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading config file:", viper.ConfigFileUsed())
	}

}
