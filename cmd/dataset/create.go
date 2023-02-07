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

// CreateCmd creates a new dataset.
var CreateCmd = &cobra.Command{
	Use:   "create '<name>' '<description>' '[\"<tag1>\", \"<tag2>\", ...]'",
	Short: "Create a new dataset.",
	Long:  `Create a new dataset within a users organization`,
	Args:  cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {

		name := args[0]
		description := args[1]
		tags := args[2]

		db, _ := config.InitializeDB()
		userSettingsStore := store.NewUserSettingsStore(db)
		userInfoStore := store.NewUserInfoStore(db)
		pennsieveClient, err := config.InitPennsieveClient(userSettingsStore, userInfoStore, nil)
		if err != nil {
			log.Fatalln("Cannot connect to Pennsieve.")
		}

		response, err := pennsieveClient.Dataset.Create(nil, name, description, tags)
		if err != nil {
			log.Error(err)
		}
		PrettyPrintCreate(response)
	},
}

func PrettyPrintCreate(ds *dataset.CreateDatasetResponse) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("New dataset created")
	t.AppendRows([]table.Row{
		{"NAME", ds.Content.Name},
		{"INT ID", ds.Content.IntID},
		{"NODE ID", ds.Content.ID},
		{"ORGANIZATION", ds.Organization},
		{"Description", ds.Content.Description},
	})

	t.Render()
}
