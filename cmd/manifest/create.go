package manifest

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"path/filepath"
)

var CreateCmd = &cobra.Command{
	Use:   "create [flags] [PATH] [...PATH]",
	Short: "Creates manifest for upload.",
	Long:  `Creates manifest for upload.`,
	Run: func(cmd *cobra.Command, args []string) {

		targetBasePath, _ := cmd.Flags().GetString("target_path")
		targetAutoPath, _ := cmd.Flags().GetBool("auto_path")
		basePath := args[0]

		if targetAutoPath && targetBasePath != "" {
			fmt.Println("Cannot set auto path and target path")
			return
		} else if targetAutoPath {
			//Get leaf directory
			targetBasePath = filepath.Base(basePath)
		}

		req := api.CreateManifestRequest{
			BasePath:       basePath,
			TargetBasePath: targetBasePath,
			Recursive:      true,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		manifestResponse, err := client.CreateManifest(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Create Manifest command: %v", err))
			return
		}

		fmt.Println("Manifest ID:", manifestResponse.ManifestId, "Message:", manifestResponse.Message)
	},
}

func init() {
	CreateCmd.Flags().StringP("target_path", "t",
		"", "Target base path in dataset.")

	CreateCmd.Flags().BoolP("auto_path", "a",
		false, "Automatically creates the base path for dataset using the last folder in the given path")

}
