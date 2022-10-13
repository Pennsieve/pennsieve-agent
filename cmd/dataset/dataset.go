package dataset

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/dataset"
	"github.com/spf13/cobra"
	"log"
	"os"
	"unicode"
)

// DatasetCmd shows the currently active dataset.
var DatasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Show the active dataset.",
	Long: `Shows the dataset that is currently active. 

Any manifests that are created will be uploaded to the active dataset.`,
	Run: func(cmd *cobra.Command, args []string) {
		showFull, _ := cmd.Flags().GetBool("full")

		db, _ := db.InitializeDB()
		userSettingsStore := store.NewUserSettingsStore(db)
		s, _ := userSettingsStore.Get()

		userInfoStore := store.NewUserInfoStore(db)

		pennsieveClient, err := api.InitPennsieveClient(userSettingsStore, userInfoStore)
		if err != nil {
			log.Fatalln("Cannot connect to Pennsieve.")
		}

		if len(s.UseDatasetId) == 0 {
			fmt.Println("\nError: No dataset specified; use 'pennsieve dataset use <node-id>' to set active dataset.")
			return
		}

		response, err := pennsieveClient.Dataset.Get(nil, s.UseDatasetId)
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
	DatasetCmd.AddCommand(FindCmd)

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

// truncateName truncates string based on max length and appends ...
func truncateName(str string, max int) string {
	lastSpaceIx := -1
	len := 0
	for i, r := range str {
		if unicode.IsSpace(r) {
			lastSpaceIx = i
		}
		len++
		if len >= max {
			if lastSpaceIx != -1 {
				return str[:lastSpaceIx] + "..."
			}
			// If here, string is longer than max, but has no spaces
			return str[:max] + "..."
		}
	}
	return str
}
