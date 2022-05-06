package manifest

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"os"
)

var ListCmd = &cobra.Command{
	Use:   "list [flags] [PATH] [...PATH]",
	Short: "Creates manifest for upload.",
	Long:  `Creates manifest for upload.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Manifest List called")

		//var uploadRecord models.UploadRecord
		//records, _ := uploadRecord.GetAll()

		req := pb.ListFilesRequest{
			ManifestId: args[0],
			Offset:     0,
			Limit:      100,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		listFilesResponse, err := client.ListFilesForManifest(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
			return
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Session ID", "Source Path", "Target Path", "S3 Key", "Status"})
		//t.SetAllowedRowLength(200)
		t.SetAutoIndex(true)
		for _, path := range listFilesResponse.File {
			t.AppendRow([]interface{}{path.SessionId, path.SourcePath, path.TargetPath, path.S3Key, path.Status})
		}

		t.Render()
	},
}

func init() {

}
