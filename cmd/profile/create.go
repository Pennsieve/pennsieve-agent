package profile

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Pennsieve profile.",
	Long:  `Creates a new Pennsieve profile which includes API-Key and API-Secret.`,
	Run: func(cmd *cobra.Command, args []string) {

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

			// Write new configuration file.
			err := viper.WriteConfig()
			if err != nil {
				fmt.Println(err)
			}
		}

	},
}

func init() {
}
