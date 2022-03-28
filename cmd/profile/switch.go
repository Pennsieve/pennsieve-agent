/*
Copyright © 2022 University of Pennsylvania <support@pennsieve.io>>

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
package profile

import (
	"fmt"
	"github.com/pennsieve/pennsieve-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var SwitchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch between user profiles.",
	Long:  `Switch between user profiles that are defined in the Pennsieve Config file.`,

	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		selectedProfile := args[0]

		// Check if profile exist
		isSet := viper.IsSet(selectedProfile + ".api_token")
		if !isSet {
			fmt.Printf("Profile %s not found\n", selectedProfile)
			return
		}

		// Profile exists, verify login and refresh token if necessary
		apiToken := viper.GetString(selectedProfile + ".api_token")
		apiSecret := viper.GetString(selectedProfile + ".api_secret")

		fmt.Println("apiKey: " + apiToken + " \napiSecret: " + apiSecret)

		client := pennsieve.NewClient()
		client.Authentication.Authenticate(apiToken, apiSecret)

		user, _ := client.User.GetUser(nil, nil)
		fmt.Println(user)

		// Update UserInfo if necessary
		if client.Credentials.IsRefreshed {
			//updateCredsInDB
			client.Credentials.IsRefreshed = false
		}

		// Store current active profile in UserSettings

	},
}

func init() {
}
