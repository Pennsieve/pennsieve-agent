package _map

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
)

var DiffCmd = &cobra.Command{
	Use:   "diff [path]",
	Short: "List local changes compared to last fetch from Pennsieve.",
	Long: `
  [BETA] This feature is in Beta mode and is currently still undergoing
  testing and optimization. 

  The 'diff' command allows users to see local changes to a Pennsieve
  mapped dataset compared to the last time the dataset was fetched from
  the Pennsieve servers. Users will be notified of ADDED, RENAMED, MOVED,
  DELETED and CHANGED files.
  `,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		exPath := args[0]

		statusRequest := api.MapDiffRequest{
			Path: exPath,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		statusResponse, err := client.GetMapDiff(context.Background(), &statusRequest)
		if err != nil {
			fmt.Println("Error calling GetMapStatus: ", err)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Path", "File Name", "Update"})
		for _, s := range statusResponse.Files {
			const maxLength = 100
			t.AppendRow([]interface{}{s.Content.Path, s.Content.Name, s.ChangeType.String()})
		}

		t.Render()

	},
}

func init() {

}
