package dataset

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	"github.com/spf13/cobra"
	"log"
)

// whoamiCmd represents the whoami command
var UseCmd = &cobra.Command{
	Use:   "use <dataset>",
	Short: "Set your current working dataset.",
	Long: `Set your current working dataset.
	
	ARGS:
    <dataset>    
            A dataset's ID or name. If omitted, the current dataset will be printed.
	`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// 1. Verify that the dataset exists
		datasetId := args[0]
		client := api.PennsieveClient
		response, err := client.Dataset.Get(nil, datasetId)
		if err != nil {
			log.Fatalln("Unknown dataset: ", datasetId)
		}

		// 2. Update UserSettings to contain dataset ID
		var userSettings models.UserSettings
		err = userSettings.UpdateActiveDataset(datasetId)
		if err != nil {
			log.Fatalln("Unable to update UserSettings:", err)
		}
		fmt.Println(response)
	},
}

func updateUserSettings(datasetId string) {

}
