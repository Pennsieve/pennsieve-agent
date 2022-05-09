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
	"github.com/spf13/cobra"
)

var ProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage Pennsieve profiles",
	Long: `Profiles are used to store user-settings. They are stored in the ~/.server/db.ini file.

`,
	Run: func(cmd *cobra.Command, args []string) {
		ShowCmd.Run(cmd, args)
	},
}

func init() {
	ProfileCmd.AddCommand(SwitchCmd)
	ProfileCmd.AddCommand(ShowCmd)
	ProfileCmd.AddCommand(SetDefaultCmd)
}
