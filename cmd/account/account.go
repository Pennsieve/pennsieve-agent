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
package account

import (
	"github.com/spf13/cobra"
)

// AccountCmd keeps the retired account commands so scripts that run them get
// a pointer to the Pennsieve app instead of "unknown command". AWS accounts
// are connected, upgraded and disconnected in the app now: its CloudFormation
// template makes the role only in the account being connected, which the
// agent's stacks (made without that account's ID) no longer satisfy.
var AccountCmd = &cobra.Command{
	Use:   "account [flags]",
	Short: "Retired: manage AWS accounts in the Pennsieve app",
	Long:  retiredLong,
	// The commands only point to the app, so they need no agent config.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
}

func init() {
	AccountCmd.AddCommand(RegisterCmd, UpdateRoleCmd, DeregisterCmd)
	// The old flags are still accepted, so an old script reaches the pointer
	// instead of failing on an unknown flag.
	for _, c := range []*cobra.Command{RegisterCmd, UpdateRoleCmd, DeregisterCmd} {
		c.Flags().String("type", "AWS", "Account Type (ignored: retired)")
		c.Flags().String("profile", "default", "CLI profile (ignored: retired)")
	}
	DeregisterCmd.Flags().Bool("force", false, "ignored: retired")
	DeregisterCmd.Flags().Bool("delete-stack", false, "ignored: retired")
}
