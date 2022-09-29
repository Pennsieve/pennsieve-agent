package manifest

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/pennsieve/pennsieve-agent/protos"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"strconv"
)

var ListCmd = &cobra.Command{
	Use:   "list [flags] <manifestId> [offset] [limit]",
	Short: "lists files for a manifest.",
	Long:  `Creates manifest for upload.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		limit := int32(100)
		offset := int32(0)

		i, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			panic(err)
		}
		manifestId := int32(i)

		if len(args) > 2 {
			i, err = strconv.ParseInt(args[2], 10, 32)
			if err != nil {
				panic(err)
			}
			offset = int32(i)
		}
		if len(args) > 1 {
			i, err = strconv.ParseInt(args[1], 10, 32)
			if err != nil {
				panic(err)
			}
			limit = int32(i)
		}

		req := pb.ListManifestFilesRequest{
			ManifestId: manifestId,
			Offset:     offset,
			Limit:      limit,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		listFilesResponse, err := client.ListManifestFiles(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err,
				fmt.Sprintf("Error: Unable to complete List Manifest command: %v", err))
			return
		}

		showFull, _ := cmd.Flags().GetBool("full")
		PrettyPrint(listFilesResponse, args[0], showFull)
	},
}

func init() {
	ListCmd.Flags().BoolP("full", "f",
		false, "Show expanded information")
}

// PrettyPrint renders a table with current userinfo to terminal
func PrettyPrint(files *protos.ListManifestFilesResponse, manifestID string, showFull bool) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle(fmt.Sprintf("Files for upload manifest: %s", manifestID))
	if showFull {
		t.AppendHeader(table.Row{"id", "Upload ID", "Source Path", "Target Path", "Status"})
		for _, path := range files.File {
			t.AppendRow([]interface{}{path.Id, path.UploadId, path.SourcePath, path.TargetPath, path.Status})
		}
	} else {
		t.AppendHeader(table.Row{"id", "Source Path", "Status"})
		for _, path := range files.File {
			t.AppendRow([]interface{}{path.Id, path.SourcePath, path.Status})
		}
	}

	t.Render()
}
