package download

import (
	"fmt"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/cmd/shared"
	pkgshared "github.com/pennsieve/pennsieve-agent/v2/pkg/shared"
	"github.com/spf13/cobra"
)

var SelectionCmd = &cobra.Command{
	Use:   "selection [selection-id] [target-folder]",
	Short: "Download files selected in Pennsieve or on Discover.",
	Long: `Download a selection made in the Pennsieve app or on Discover, by the id it shows (sel_...), to the selected folder (default: the current folder).

  pennsieve download selection sel_l2uw6ebgbzymzzusdedg64i5wm ./study

A selection lasts two days. It downloads with your own access: you get the files you're allowed to download.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		selectionId := args[0]
		folder := "."
		if len(args) > 1 {
			folder = args[1]
		}
		absPath, err := shared.GetAbsolutePath(folder)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
			return
		}
		force, _ := cmd.Flags().GetBool("force")

		resp, ok := requestDownload(&api.DownloadRequest{
			Type: api.DownloadRequest_SELECTION,
			Data: &api.DownloadRequest_Selection{Selection: &api.DownloadSelectionRequest{
				SelectionId: selectionId, TargetFolder: absPath, Force: force,
			}},
		})
		if !ok {
			return
		}
		of := "the selection"
		if resp.PublicDatasetId != 0 {
			of = fmt.Sprintf("dataset %d, version %d", resp.PublicDatasetId, resp.PublicVersion)
		}
		fmt.Printf("Downloading %d files (%s) of %s to %s\n",
			resp.FileCount, pkgshared.HumanBytes(resp.TotalBytes), of, absPath)
		fmt.Println("Follow progress with: pennsieve agent subscribe")
		fmt.Println("Cancel with: pennsieve download cancel " + selectionId)
	},
}

func init() {
	SelectionCmd.Flags().Bool("force", false,
		"Download even when the target folder's disk has too little free space")
}
