package manifest

import (
	"context"
	"fmt"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var AddCmd = &cobra.Command{
	Use:   "add [manifest-id] [PATH]",
	Short: "Add to manifest for upload.",
	Long:  `Add to manifest for upload.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		manifestId := args[0]
		localBasePath := args[1]

		targetBasePath, _ := cmd.Flags().GetString("target_path")
		recursive, _ := cmd.Flags().GetBool("recursive")

		req := pb.AddToManifestRequest{
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

		client := pb.NewAgentClient(conn)
		manifestResponse, err := client.AddToManifest(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
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
