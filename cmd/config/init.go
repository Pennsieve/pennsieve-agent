package config

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

var InitCmd = &cobra.Command{
	Use:   "init <config file path>",
	Short: "Create a new Pennsieve Profile at a specified location.",
	Long: `Create a new Pennsieve configuration file and add profile with API Key/secret.

NOTE: This method ignores the globally defined "config" parameter. You set the file path for 
the new configuration file as the first argument for the method.

NOTE: When invoking the Pennsieve CLI, when you want to use a different config.ini file than the default file, you
need to specify the config.ini file location with every command by passing in the "config" parameter. `,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		configFile := args[0]
		forceCreate, _ := cmd.Flags().GetBool("force")
		profileName, _ := cmd.Flags().GetString("profile")
		apiToken, _ := cmd.Flags().GetString("api_token")
		apiSecret, _ := cmd.Flags().GetString("api_secret")

		if len(configFile) == 0 {

		}

		if strings.HasPrefix(configFile, "~/") {
			dirname, _ := os.UserHomeDir()
			configFile = filepath.Join(dirname, configFile[2:])
		}

		// Check if file already exists and confirm user wants to replace.
		_, err := os.Stat(configFile)
		if err == nil {
			if !forceCreate {
				fmt.Println("Existing configuration file found at:", configFile)
				fmt.Printf("\nWould you like to overwrite your existing configuration? (y/n): ")

				response := ""
				fmt.Scanln(&response)

				if response != "y" {
					fmt.Println("Cancelling action.")
					return
				}

				os.Remove(configFile)
			} else {
				os.Remove(configFile)
			}
		} else {
			fmt.Println(err)
		}

		viper.SetConfigFile(configFile)
		viper.AddConfigPath(configFile)

		configPath := filepath.Dir(configFile)

		// Ensure folder is created
		os.MkdirAll(configPath, os.ModePerm)

		viper.SetConfigType("ini")
		viper.Set("agent.port", "9000")
		viper.Set("agent.upload_workers", "10")    // Number of concurrent files during upload
		viper.Set("agent.upload_chunk_size", "32") // Upload chunk-size in MB
		viper.Set(fmt.Sprintf("%s.api_token", profileName), apiToken)
		viper.Set(fmt.Sprintf("%s.api_secret", profileName), apiSecret)
		viper.Set("global.default_profile", profileName)

		// Write new configuration file.
		err = viper.WriteConfig()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("New configuration file created: ", viper.ConfigFileUsed())

	},
}

func init() {

	InitCmd.MarkFlagRequired("config")

	InitCmd.Flags().String("profile",
		"user", "Profile name to be associated with provided api key/secret")

	InitCmd.Flags().String("api_token",
		"", "Target base path in dataset.")

	InitCmd.Flags().String("api_secret",
		"", "Target base path in dataset.")

	InitCmd.Flags().BoolP("force", "f",
		false, "Force creation of config file.")

}
