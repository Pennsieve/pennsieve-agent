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
	Short: "Switch profile",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		selectedProfile, _ := cmd.Flags().GetString("profile")
		apiToken := viper.GetString(selectedProfile + ".api_token")
		apiSecret := viper.GetString(selectedProfile + ".api_secret")

		fmt.Println("apiKey: " + apiToken + " \napiSecret: " + apiSecret)

		//ps := pennsieve_go.Pennsieve{
		//	ApiToken:  apiToken,
		//	ApiSecret: apiSecret,
		//	Profile:   selectedProfile,
		//}

		client := pennsieve.NewClient()
		client.Authentication.Authenticate(apiToken, apiSecret)

		user, _ := client.User.GetUser(nil, nil)
		fmt.Println(user)

		if client.Credentials.IsRefreshed {
			//updateCredsInDB
			client.Credentials.IsRefreshed = false
		}

		fmt.Printf("Organization Node Id: %s\n", client.OrganizationNodeId)
		//_, err := config.InitializeDB()
		//if err != nil {
		//	log.Println("Driver creation failed", err.Error())
		//} else {
		//	// Run all migrations
		//	migrations.Run()
		//
		//}
	},
}

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// whoamiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	SwitchCmd.PersistentFlags().StringP("profile", "p", "", "Set Pennsieve profile to use")
	//viper.BindPFlag("activeProfile", SwitchCmd.PersistentFlags().Lookup("profile"))

}
