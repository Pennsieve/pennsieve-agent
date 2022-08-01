package config

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
)

var WizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Create a new config file using the configuration wizard.",
	Long: `Create a new config file using the configuration wizard.

Use the Pennsieve configuration wizard to create a new Pennsieve Configuration file and add an initial set of 
API credentials. 

NOTE: This method will remove any existing configuration file if it exists and previously defined API-Keys and secrets 
will not be recoverable. Use the 'pennsieve profile create' function to add profiles to an existing configuration file.
`,
	Run: func(cmd *cobra.Command, args []string) {

		// Create .Pennsieve folder if it does not exist
		home, err := os.UserHomeDir()
		pennsieveFolder := filepath.Join(home, ".pennsieve")
		configFile := filepath.Join(pennsieveFolder, "config.ini")

		// Check if file already exists and confirm user wants to replace.
		_, err = os.Stat(configFile)
		if err == nil {
			fmt.Println("Existing configuration file found at:", configFile)
			fmt.Printf("\nWould you like to overwrite your existing configuration? (y/n): ")

			response := ""
			fmt.Scanln(&response)

			if response != "y" {
				return
			}

			os.Remove(configFile)
		}

		fmt.Println("\nCreating new configuration file at", configFile)

		// Create './pennsieve' folder if it does not exist.
		if _, err := os.Stat(pennsieveFolder); errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(pennsieveFolder, os.ModePerm); err != nil {
				log.Fatal(err)
			}
		}

		var profileName string
		fmt.Println("\nCreate new profile:")
		fmt.Printf("   Profile name [user]: ")
		fmt.Scanln(&profileName)

		if len(profileName) == 0 {
			profileName = "user"
		}

		var apiToken string
		fmt.Printf("   API token: ")
		fmt.Scanln(&apiToken)

		var apiSecret string
		fmt.Printf("   API secret: ")
		fmt.Scanln(&apiSecret)

		fmt.Printf("Creating new profile: '%s'\n", profileName)

		fmt.Printf("Continue and write changes? (y/n) ")
		response := ""
		fmt.Scanln(&response)

		if response == "y" {
			viper.Set(fmt.Sprintf("%s.api_token", profileName), apiToken)
			viper.Set(fmt.Sprintf("%s.api_secret", profileName), apiSecret)
			viper.Set("global.default_profile", profileName)

			// Write new configuration file.
			err = viper.SafeWriteConfig()
			if err != nil {
				fmt.Println(err)
			}
		}
	},
}
