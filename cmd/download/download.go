package download

import (
	"github.com/spf13/cobra"
)

var DownloadCmd = &cobra.Command{
	Use:   "download [command] [...Args]",
	Short: "Download a package, a dataset, a published dataset or a selection.",
	Long: `
  You can download packages and datasets using one of the 
  respective subcommands. Files will be downloaded by the 
  agent in the background and you can check progress by running 
  the agent subscriber.`,
}

func init() {

	DownloadCmd.PersistentFlags().BoolP("presigned", "u",
		false, "Return presigned url (default false) ")

	DownloadCmd.AddCommand(PackageCmd)
	DownloadCmd.AddCommand(DatasetCmd)
	DownloadCmd.AddCommand(PublicCmd)
	DownloadCmd.AddCommand(SelectionCmd)
	DownloadCmd.AddCommand(CancelCmd)
}
