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
	"unicode"
)

var ManifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Lists upload sessions.",
	Long: `Renders a list of upload manifests and their current status. 

This list includes only upload manifests that are initiated from the current machine.`,
	Run: func(cmd *cobra.Command, args []string) {

		req := pb.ListManifestsRequest{}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		manifestResponse, err := client.ListManifests(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
			return
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Upload Manifest", "User Name", "Organization Name", "Dataset ID", "Status", "nodeId"})
		for _, s := range manifestResponse.Manifests {
			const maxLength = 100
			dsName := trimName(s.DatasetName, maxLength)
			t.AppendRow([]interface{}{s.Id, s.UserName, s.OrganizationName, dsName, s.Status, s.NodeId})
		}

		t.Render()
	},
}

func init() {
	ManifestCmd.AddCommand(ListCmd)
	ManifestCmd.AddCommand(CreateCmd)
	ManifestCmd.AddCommand(AddCmd)
	ManifestCmd.AddCommand(RemoveCmd)
	ManifestCmd.AddCommand(DeleteCmd)
	ManifestCmd.AddCommand(SyncCmd)
	ManifestCmd.AddCommand(ResetCmd)
}

func trimName(str string, max int) string {
	lastSpaceIx := -1
	len := 0
	for i, r := range str {
		if unicode.IsSpace(r) {
			lastSpaceIx = i
		}
		len++
		if len >= max {
			if lastSpaceIx != -1 {
				return str[:lastSpaceIx] + "..."
			}
			return str[:max]
		}
	}
	return str
}
