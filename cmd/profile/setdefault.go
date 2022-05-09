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
package profile

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var SetDefaultCmd = &cobra.Command{
	Use:   "set-default",
	Short: "Update the default profile",
	Long:  `Stores a default profile in the Pennsieve db.ini file`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		profile := args[0]

		// Check if profile exists --> Should have an API Token
		isSet := viper.IsSet(profile + ".api_token")

		if isSet {
			viper.Set("global.default_profile", profile)
			viper.WriteConfig()
			fmt.Println("Default profile set to:", profile)
		} else {
			fmt.Printf("No profile with name %s exists.\n", profile)
		}
	},
}

func init() {
}
