package manifest

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/spf13/cobra"
	"os"
)

var ListCmd = &cobra.Command{
	Use:   "list [flags] [PATH] [...PATH]",
	Short: "Creates manifest for upload.",
	Long:  `Creates manifest for upload.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Manifest List called")

		var uploadRecord models.UploadRecord
		records, _ := uploadRecord.GetAll()

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Session ID", "Source Path", "Target Path"})
		//t.SetAllowedRowLength(200)
		t.SetAutoIndex(true)
		for _, path := range records {
			t.AppendRow([]interface{}{path.SessionID, path.SourcePath, path.TargetPath})
		}

		t.Render()
	},
}

func init() {

}
