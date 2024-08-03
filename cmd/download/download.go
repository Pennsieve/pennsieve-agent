package download

import (
	"github.com/spf13/cobra"
)

var DownloadCmd = &cobra.Command{
	Use:   "download [flags] [PACKAGE_ID]",
	Short: "Downloads a file.",
	Long:  `Use this function to download a file.`,

	Run: func(cmd *cobra.Command, args []string) {
		PackageCmd.Run(cmd, args)
	},
}

func init() {

	DownloadCmd.PersistentFlags().BoolP("presigned", "u",
		false, "Return presigned url (default false) ")

	DownloadCmd.AddCommand(PackageCmd)
	DownloadCmd.AddCommand(CancelCmd)
}
