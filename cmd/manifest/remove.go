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
	"os"
)

var RemoveCmd = NewRemoveCmd()

// NewRemoveCmd returns a new manifest remove sub-command.
// Useful for testing since the re-use of a global *cobra.Command variable in tests causes
// problems with flag values being retained, and so one test can pollute another when run in parallel.
func NewRemoveCmd() *cobra.Command {
	manifestIdFlag := "manifest_id"

	cmd := &cobra.Command{
		Use:   "remove -m MANIFEST-ID SOURCE-PATH-PREFIX",
		Short: "Removes files from an existing manifest.",
		Long: `Removes files from an existing manifest.
This command will remove any files from the manifest with id MANIFEST-ID with status LOCAL and with a source path that starts with SOURCE-PATH-PREFIX.
If a file in the manifest has a source path starting with SOURCE-PATH-PREFIX and status REGISTERED, then the status will be updated to Removed.`,
		// this is the one positional arg, the source path
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			manifestId, err := cmd.Flags().GetInt32(manifestIdFlag)
			if err != nil {
				printErr(cmd, err.Error())
				os.Exit(1)
			}
			printOut(cmd, fmt.Sprintf("manifest id: %d", manifestId))

			// Args field in this Command ensures we only get here if len(args) == 1
			sourcePath := args[0]

			printOut(cmd, fmt.Sprintf("source path prefix: %s", sourcePath))

			req := api.RemoveFromManifestRequest{
				ManifestId: manifestId,
				RemovePath: sourcePath,
			}

			port := viper.GetString("agent.port")

			conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				printOut(cmd, fmt.Sprintf("Error connecting to GRPC Server: %v", err))
				return
			}
			defer conn.Close()

			client := api.NewAgentClient(conn)
			manifestResponse, err := client.RemoveFromManifest(context.Background(), &req)
			if err != nil {
				shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Remove Manifest command: %v", err))
				return
			}

			printOut(cmd, manifestResponse.Status)
		},
	}

	cmd.Flags().Int32P(manifestIdFlag, "m",
		0, "Manifest id")

	if err := cmd.MarkFlagRequired(manifestIdFlag); err != nil {
		printErr(cmd, err.Error())
	}

	return cmd

}

func printOut(cmd *cobra.Command, msg string) {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), msg)
}

func printErr(cmd *cobra.Command, msg string) {
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), msg)
}
