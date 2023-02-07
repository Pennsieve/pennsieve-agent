package dataset

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pennsieve/pennsieve-agent/pkg/config"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/dataset"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

// FindCmd lists datasets based on input query
var FindCmd = &cobra.Command{
	Use:   "find '<query>'",
	Short: "Find datasets",
	Args:  cobra.MinimumNArgs(1),
	Long: `Lists datasets based on a query.

Search is fuzzy and returns datasets based on matches in:
- title
- authors
- tags
- description
`,
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		limit, _ := cmd.Flags().GetInt("limit")

		db, _ := config.InitializeDB()
		userSettingsStore := store.NewUserSettingsStore(db)
		userInfoStore := store.NewUserInfoStore(db)
		pennsieveClient, err := config.InitPennsieveClient(userSettingsStore, userInfoStore, nil)
		if err != nil {
			log.Fatalln("Cannot connect to Pennsieve.")
		}

		response, err := pennsieveClient.Dataset.Find(nil, limit, query)
		if err != nil {
			log.Error(err)
		}

		PrettyPrintFind(response)
	},
}

func init() {
	FindCmd.Flags().IntP("limit", "l",
		100, "Limit")
}

func PrettyPrintFind(ds *dataset.ListDatasetResponse) {
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
