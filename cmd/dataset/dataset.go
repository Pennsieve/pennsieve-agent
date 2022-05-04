package dataset

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/spf13/cobra"
)

// whoamiCmd represents the whoami command
var DatasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Set your current working dataset.",
	Long:  `Set your current working dataset.`,
	Run: func(cmd *cobra.Command, args []string) {
		var userSettings models.UserSettings
		s, _ := userSettings.Get()

		fmt.Println("Currently active dataset:", s.UseDatasetId)
	},
}

func init() {
	DatasetCmd.AddCommand(UseCmd)
	DatasetCmd.AddCommand(ListCmd)

}
