package _map

import (
	"github.com/spf13/cobra"
)

var PushCmd = &cobra.Command{
	Use:   "push [target_path]",
	Short: "Push local changes to the remote Pennsieve Dataset",
	Long: `
  [BETA] This feature is in Beta mode and is currently still undergoing
  testing and optimization.

  Currently unimplemented

  `,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		//target_path := args[0]
		//
		//// Check and make path absolute
		//absPath, err := shared.GetAbsolutePath(target_path)
		//if err != nil {
		//    fmt.Println(err)
		//    shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
		//    return
		//}
		//
		//pullRequest := api.PullRequest{
		//    Path: absPath,
		//}
		//
		//port := viper.GetString("agent.port")
		//conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		//if err != nil {
		//    fmt.Println("Error connecting to GRPC Server: ", err)
		//    return
		//}
		//defer conn.Close()
		//
		//client := api.NewAgentClient(conn)
		//pullResponse, err := client.Pull(context.Background(), &pullRequest)
		//if err != nil {
		//    fmt.Println(err)
		//    shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Fetch command: %v", err))
		//    return
		//}
		//if pullResponse.Status == "Success" {
		//    fmt.Println("success")
		//} else {
		//    fmt.Println("Unable to request pull command: ", pullResponse.Status)
		//    log.Errorf("Unable to request pull command: %v", pullResponse.Status)
		//}

	},
}

func init() {

}
