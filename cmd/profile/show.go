/*
Copyright © 2022 University of Pennsylvania <support@pennsieve>>

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
	"github.com/pennsieve/pennsieve-agent/api"
	"github.com/pennsieve/pennsieve-go"
	"github.com/spf13/cobra"
)

var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Shows current profile",
	Long:  `Shows current profile`,
	Run: func(cmd *cobra.Command, args []string) {
		client := pennsieve.NewClient()
		activeUser, _ := api.GetActiveUser(client)
		fmt.Println("Current profile: ", activeUser.Profile)
	},
}

func init() {
}
