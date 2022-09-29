package dataset

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pennsieve/pennsieve-agent/cmd/config"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/dataset"
	"github.com/spf13/cobra"
	"log"
	"os"
)

// ListCmd renders a list of datasets for a user.
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all datasets.",
	Long:  `List all datasets in a Pennsieve Workspace that are accessible to the user.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.InitDB()
	},
	Run: func(cmd *cobra.Command, args []string) {
		offset, _ := cmd.Flags().GetInt("offset")
		limit, _ := cmd.Flags().GetInt("limit")

		client := api.PennsieveClient
		response, err := client.Dataset.List(nil, limit, offset)
		if err != nil {
			log.Println(err)
		}

		PrettyPrintList(response)
	},
}

func init() {
	ListCmd.Flags().IntP("offset", "o",
		0, "Offset (default 0) ")

	ListCmd.Flags().IntP("limit", "l",
		100, "Limit")

}

func PrettyPrintList(ds *dataset.ListDatasetResponse) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("Datasets")
	t.AppendHeader(table.Row{"NAME", "Node ID", "Integer ID"})
	for _, d := range ds.Datasets {
		truncatedName := truncateName(d.Content.Name, 50)
		t.AppendRow([]interface{}{truncatedName, d.Content.ID, d.Content.IntID})
	}

	t.Render()
}
