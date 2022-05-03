package manifest

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list [flags] [PATH] [...PATH]",
	Short: "Creates manifest for upload.",
	Long:  `Creates manifest for upload.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Manifest List called")

		var uploadRecord models.UploadRecord
		records, _ := uploadRecord.GetAll()

		fmt.Println(records)

	},
}

func init() {

}
