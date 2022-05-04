package dataset

import (
	"github.com/spf13/cobra"
)

// whoamiCmd represents the whoami command
var DatasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Set your current working dataset.",
	Long:  `Set your current working dataset.`,
}

func init() {
	DatasetCmd.AddCommand(UseCmd)
	DatasetCmd.AddCommand(ListCmd)

}
