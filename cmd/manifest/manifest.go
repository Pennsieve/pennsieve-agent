package manifest

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/spf13/cobra"
	"os"
)

var ManifestCmd = &cobra.Command{
	Use:   "manifest [flags] [PATH] [...PATH]",
	Short: "Lists upload sessions.",
	Long:  `Creates manifest for upload.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Manifest called")

		var uploadSession models.UploadSession
		sessions, _ := uploadSession.GetAll()

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Session ID", "User ID", "Organization ID", "Dataset ID", "Status"})
		//t.SetAllowedRowLength(200)
		for _, s := range sessions {
			t.AppendRow([]interface{}{s.SessionId, s.UserId, s.OrganizationId, s.DatasetId, s.Status})
		}

		t.Render()

	},
}

func init() {
	ManifestCmd.AddCommand(ListCmd)
	ManifestCmd.AddCommand(CreateCmd)

}
