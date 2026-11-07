package account

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

const retiredLong = `AWS accounts are managed in the Pennsieve app, under Your compute resources:
  - Connect AWS account: connect an account.
  - Upgrade or Update on an account: move its role to Pennsieve's latest template.
  - Disconnect on the account's page: remove it.
The app opens your AWS console on Pennsieve's CloudFormation template, which creates
the role only in the account you're connecting. These commands no longer do anything.`

// retired is a command that only says where its job is done now, and fails,
// so a script that runs it stops there.
func retired(use, short, inTheApp string) *cobra.Command {
	return &cobra.Command{
		Use:          use,
		Short:        "Retired: " + short,
		Long:         retiredLong,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.ErrOrStderr(), "pennsieve account %s is retired. %s\n", use, inTheApp)
			return errors.New("retired: use the Pennsieve app")
		},
	}
}

var (
	RegisterCmd = retired("register", "connect an AWS account in the Pennsieve app",
		"Connect the account in the Pennsieve app: Your compute resources → Connect AWS account.")
	UpdateRoleCmd = retired("update-role", "update an account's role in the Pennsieve app",
		"Update the account's role in the Pennsieve app: Your compute resources → Upgrade (or Update) on the account.")
	DeregisterCmd = retired("deregister", "disconnect an AWS account in the Pennsieve app",
		"Disconnect the account in the Pennsieve app: Your compute resources → the account's page → Disconnect.")
)
