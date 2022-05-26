package manifest

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pennsieve/pennsieve-agent/protos"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"os"
	"strconv"
)

var ListCmd = &cobra.Command{
	Use:   "list [flags] [PATH] [...PATH]",
	Short: "Creates manifest for upload.",
	Long:  `Creates manifest for upload.`,
	Run: func(cmd *cobra.Command, args []string) {

		limit := int64(100)
		offset := int64(100)
		if len(args) > 2 {
			offset, _ = strconv.ParseInt(args[1], 10, 32)
		}
		if len(args) > 1 {
			limit, _ = strconv.ParseInt(args[2], 10, 32)
		}

		req := pb.ListManifestFilesRequest{
			ManifestId: args[0],
			Offset:     int32(offset),
			Limit:      int32(limit),
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
			st := status.Convert(err)
			fmt.Println(st.Message())
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
func PrettyPrint(files *protos.ListFilesResponse, manifestID string, showFull bool) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle(fmt.Sprintf("Files for upload session: %s", manifestID))
	if showFull {
		t.AppendHeader(table.Row{"id", "Source Path", "Target Path", "Status"})
		for _, path := range files.File {
			t.AppendRow([]interface{}{path.Id, path.SourcePath, path.TargetPath, path.Status})
		}
	} else {
		t.AppendHeader(table.Row{"id", "Source Path", "Status"})
		for _, path := range files.File {
			t.AppendRow([]interface{}{path.Id, path.SourcePath, path.Status})
		}
	}

	t.Render()
}
