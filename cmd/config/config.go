// Package db /*
package config

import (
	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the current Pennsieve configuration file.",
	Long:  `Show the current Pennsieve configuration file.`,
	Run: func(cmd *cobra.Command, args []string) {

		//TODO Show current config.ini file.

	},
}

func init() {
	ConfigCmd.AddCommand(WizardCmd)
}
