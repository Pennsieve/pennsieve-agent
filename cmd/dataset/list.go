package dataset

import "github.com/spf13/cobra"

// whoamiCmd represents the whoami command
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "Set your current working dataset.",
	Long:  `Set your current working dataset.`,
	Run: func(cmd *cobra.Command, args []string) {
		//client := pennsieve.NewClient()
		//activeUser, _ := api.GetActiveUser(client)
		//showFull, _ := cmd.Flags().GetBool("full")

	},
}

func init() {

}
