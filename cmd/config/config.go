// Package config /*
package config

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the current Pennsieve configuration file.",
	Long: `Show the current Pennsieve configuration file.

The Pennsieve configuration file contains information for the Pennsieve agent to authenticate with the Pennsieve Platform.

It is usually located in '~/.pennsieve/config.ini' and contains profiles, api-keys and secrets associated with a 
particular user. Do not share your config.ini file as API-Keys and secrets are private to individual users.
`,
	Run: func(cmd *cobra.Command, args []string) {

		home, err := os.UserHomeDir()
		pennsieveFolder := filepath.Join(home, ".pennsieve")
		configFile := filepath.Join(pennsieveFolder, "config.ini")
		data, err := ioutil.ReadFile(configFile)
		if err != nil {
			fmt.Println("Unable to render Pennsieve configuration file.")
			os.Exit(1)
		}
		fmt.Println(string(data))

	},
}

func init() {
	ConfigCmd.AddCommand(WizardCmd)
}
