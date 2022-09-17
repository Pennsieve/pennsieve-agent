package dataset

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pennsieve/pennsieve-agent/cmd/config"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/dataset"
	"github.com/spf13/cobra"
	"log"
	"os"
)

// DatasetCmd shows the currently active dataset.
var DatasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Show the active dataset.",
	Long: `Shows the dataset that is currently active. 

Any manifests that are created will be uploaded to the active dataset.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.InitDB()
	},
	Run: func(cmd *cobra.Command, args []string) {
		showFull, _ := cmd.Flags().GetBool("full")

		var userSettings models.UserSettings
		s, _ := userSettings.Get()

		client := api.PennsieveClient
		response, err := client.Dataset.Get(nil, s.UseDatasetId)
		if err != nil {
			log.Println(err)
			log.Fatalln("Unknown dataset: ", s.UseDatasetId)
		}

		PrettyPrint(response, showFull)
	},
}

func init() {
	DatasetCmd.AddCommand(UseCmd)
	DatasetCmd.AddCommand(ListCmd)

	DatasetCmd.Flags().BoolP("full", "f",
		false, "Show expanded information")

}

func PrettyPrint(ds *dataset.GetDatasetResponse, showFull bool) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("Active dataset")
	t.AppendRows([]table.Row{
		{"NAME", ds.Content.Name},
		{"INT ID", ds.Content.IntID},
		{"NODE ID", ds.Content.ID},
		{"ORGANIZATION", ds.Organization},
	})
	if showFull {
		t.AppendRows([]table.Row{
			{"INT ID", ds.Content.IntID},
			{"DESCRIPTION", ds.Content.Description},
		})
	}
	t.Render()
}
