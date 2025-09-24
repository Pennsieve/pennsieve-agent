package manifest

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var RemoveCmd = NewRemoveCmd()

// NewRemoveCmd returns a new manifest remove sub-command.
// Useful for testing since the re-use of a global ManifestCmd variable in tests causes
// problems with flag values being retained, and so one test can pollute another when run in parallel.
func NewRemoveCmd() *cobra.Command {
	manifestIdFlag := "manifest_id"

	cmd := &cobra.Command{
		Use:   "remove -m MANIFEST-ID SOURCE-PATH",
		Short: "Removes a file from an existing manifest.",
		Long:  `Removes a file from an existing manifest.`,
		// this is the one positional arg, the source path
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			manifestId, err := cmd.Flags().GetInt32(manifestIdFlag)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println("manifest id:", manifestId)

			// Args field in this Command ensures we only get here if len(args) == 1
			sourcePath := args[0]

			fmt.Println("source path:", sourcePath)

			req := api.RemoveFromManifestRequest{
				ManifestId: manifestId,
				RemovePath: sourcePath,
			}

			port := viper.GetString("agent.port")

			conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				fmt.Println("Error connecting to GRPC Server: ", err)
				return
			}
			defer conn.Close()

			client := api.NewAgentClient(conn)
			manifestResponse, err := client.RemoveFromManifest(context.Background(), &req)
			if err != nil {
				shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Remove Manifest command: %v", err))
				return
			}

			fmt.Println(manifestResponse.Status)
		},
	}

	cmd.Flags().Int32P(manifestIdFlag, "m",
		0, "Manifest id")

	if err := cmd.MarkFlagRequired(manifestIdFlag); err != nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err)
	}

	return cmd

}
