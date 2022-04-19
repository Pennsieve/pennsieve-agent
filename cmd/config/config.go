// Package config /*
package config

import (
	"fmt"
	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Methods for configuring the Pennsieve Agent",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("config called")

		//TODO run profile list here

	},
}

func init() {
	ConfigCmd.AddCommand(InitCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// whoamiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// whoamiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
