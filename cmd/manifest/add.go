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
	"strconv"
)

var AddCmd = &cobra.Command{
	Use:   "add [manifest-id] [PATH]",
	Short: "Add to manifest for upload.",
	Long:  `Add to manifest for upload.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		i, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			panic(err)
		}
		manifestId := int32(i)

		localBasePath := args[1]

		targetBasePath, _ := cmd.Flags().GetString("target_path")
		recursive, _ := cmd.Flags().GetBool("recursive")

		req := api.AddToManifestRequest{
			ManifestId:     manifestId,
			BasePath:       localBasePath,
			TargetBasePath: targetBasePath,
			Recursive:      recursive,
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		manifestResponse, err := client.AddToManifest(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Add To Manifest command: %v", err))
			return
		}

		fmt.Println(manifestResponse.Status)
	},
}

func init() {
	AddCmd.Flags().StringP("target_path", "t",
		"", "Target base path in dataset.")

	AddCmd.Flags().BoolP("recursive", "r",
		true, "Set indexing to be recursive")
}
