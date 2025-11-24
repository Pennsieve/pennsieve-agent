package _map

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var PushCmd = &cobra.Command{
	Use:   "push [target_path]",
	Short: "Push local changes to the remote Pennsieve Dataset",
	Long: `
  [BETA] This feature is in Beta mode and is currently still undergoing
  testing and optimization.

  Push identifies new files in your local mapped dataset and uploads them
  to Pennsieve while preserving the directory structure.
  `,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Determine the target folder
		var folder string
		if len(args) > 0 {
			folder = args[0]
		} else {
			folder = "."
		}

		// Check and make path absolute
		absPath, err := shared.GetAbsolutePath(folder)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
			return
		}

		// Connect to the agent server
		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)

		// Show diff summary for added files
		newFiles, err := displayAddedFiles(client, absPath)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, "Error: Unable to calculate diff before push")
			return
		}
		if newFiles {
			proceed, err := confirmPush()
			if err != nil {
				fmt.Println(err)
				shared.HandleAgentError(err, "Error: unable to confirm push")
				return
			}
			if !proceed {
				fmt.Println("Push canceled.")
				return
			}
		} else {
			fmt.Println("No new local files detected. Nothing to push.")
			return
		}

		// Create a push request
		pushRequest := api.PushRequest{
			Path: absPath,
		}

		pushResponse, err := client.Push(context.Background(), &pushRequest)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Push command: %v", err))
			return
		}

		fmt.Println(pushResponse.Status)
	},
}

func init() {

}

func displayAddedFiles(client api.AgentClient, path string) (bool, error) {
	diffResp, err := client.GetMapDiff(context.Background(), &api.MapDiffRequest{Path: path})
	if err != nil {
		return false, err
	}

	var added []*api.PackageStatus
	for _, status := range diffResp.GetFiles() {
		if status.GetChangeType() == api.PackageStatus_ADDED {
			added = append(added, status)
		}
	}

	if len(added) == 0 {
		return false, nil
	}

	fmt.Println("The following files will be pushed:")
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Path", "File Name", "Update"})
	for _, status := range added {
		content := status.GetContent()
		if content == nil {
			continue
		}
		const maxDisplay = 100
		pathEntry := content.GetPath()
		if len(pathEntry) > maxDisplay {
			pathEntry = pathEntry[:maxDisplay]
		}
		nameEntry := content.GetName()
		if len(nameEntry) > maxDisplay {
			nameEntry = nameEntry[:maxDisplay]
		}
		t.AppendRow(table.Row{pathEntry, nameEntry, status.GetChangeType().String()})
	}
	t.Render()

	return true, nil
}

func confirmPush() (bool, error) {
	fmt.Print("Proceed with push? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("unable to read confirmation: %w", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return false, nil
	}

	return true, nil
}
