package download

import (
	"context"
	"fmt"
	"strconv"

	api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
	"github.com/pennsieve/pennsieve-agent/v2/cmd/shared"
	pkgshared "github.com/pennsieve/pennsieve-agent/v2/pkg/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var PublicCmd = &cobra.Command{
	Use:   "public [dataset-id] [target-folder]",
	Short: "Download a published (Discover) dataset.",
	Long: `Download a published dataset from Pennsieve Discover to the selected folder: the latest version, or the one given with --version.

Use --path to download only some files or folders of the version, for example:

  pennsieve download public 5347 ./data --path files/session1 --path README.md

Embargoed versions download only when you have access to them. Files are checked against the checksums recorded when they were published.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		datasetId, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || datasetId <= 0 {
			fmt.Println("Error: the dataset id is the published dataset's number, for example 5347")
			return
		}
		absPath, err := shared.GetAbsolutePath(args[1])
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
			return
		}
		version, _ := cmd.Flags().GetInt32("version")
		paths, _ := cmd.Flags().GetStringSlice("path")
		force, _ := cmd.Flags().GetBool("force")

		resp, ok := requestDownload(&api.DownloadRequest{
			Type: api.DownloadRequest_PUBLIC,
			Data: &api.DownloadRequest_Public{Public: &api.DownloadPublicRequest{
				DatasetId: datasetId, Version: version, TargetFolder: absPath, Paths: paths, Force: force,
			}},
		})
		if !ok {
			return
		}
		fmt.Printf("Downloading %d files (%s) of dataset %d, version %d, to %s\n",
			resp.FileCount, pkgshared.HumanBytes(resp.TotalBytes), resp.PublicDatasetId, resp.PublicVersion, absPath)
		fmt.Println("Follow progress with: pennsieve agent subscribe")
		fmt.Printf("Cancel with: pennsieve download cancel %d\n", resp.PublicDatasetId)
	},
}

// requestDownload asks the agent to start a download, and prints why when
// it can't: the reason once, as the agent or download-service gave it.
func requestDownload(req *api.DownloadRequest) (*api.DownloadResponse, bool) {
	port := viper.GetString("agent.port")
	conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println("Error connecting to GRPC Server: ", err)
		return nil, false
	}
	defer conn.Close()

	resp, err := api.NewAgentClient(conn).Download(context.Background(), req)
	if err != nil {
		shared.HandleAgentError(err, "Error: "+status.Convert(err).Message())
		return nil, false
	}
	if resp.Status != "Success" {
		fmt.Println("Unable to request download command: ", resp.Status)
		return nil, false
	}
	return resp, true
}

func init() {
	PublicCmd.Flags().Int32("version", 0, "The version to download (default: the latest)")
	PublicCmd.Flags().StringSlice("path", nil,
		"Download only these files or folders of the version (repeat or separate with commas)")
	PublicCmd.Flags().Bool("force", false,
		"Download even when the target folder's disk has too little free space")
}
