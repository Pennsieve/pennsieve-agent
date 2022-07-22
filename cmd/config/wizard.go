package config

import (
	"github.com/spf13/cobra"
)

var WizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Create a new config file using the configuration wizard.",
	Long:  `Create a new config file using the configuration wizard.`,
	Run: func(cmd *cobra.Command, args []string) {

		//TODO run profile list here

	},
}

func init() {

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// whoamiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// whoamiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
