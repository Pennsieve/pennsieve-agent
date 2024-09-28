package _map

import (
	"github.com/spf13/cobra"
)

var FetchCmd = &cobra.Command{
	Use:   "fetch [dataset_id] [target_path]",
	Short: "Fetch remote state to locally mapped dataset",
	Long: `
  [BETA] This feature is in Beta mode and is currently still undergoing
  testing and optimization.

  Currently unimplemented
  `,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		//datasetId := args[0]
		//
		//folder := args[1]
		//
		//// Check and make path absolute
		//absPath, err := shared.GetAbsolutePath(folder)
		//if err != nil {
		//	fmt.Println(err)
		//	shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
		//	return
		//}
		//
		//fetchRequest := api.FetchRequest{
		//	DatasetId:    datasetId,
		//	TargetFolder: absPath,
		//}
		//
		//port := viper.GetString("agent.port")
		//conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		//if err != nil {
		//	fmt.Println("Error connecting to GRPC Server: ", err)
		//	return
		//}
		//defer conn.Close()
		//
		//client := api.NewAgentClient(conn)
		//fetchResponse, err := client.Fetch(context.Background(), &fetchRequest)
		//if err != nil {
		//	fmt.Println(err)
		//	shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Fetch command: %v", err))
		//	return
		//}
		//if fetchResponse.Status == "Success" {
		//	fmt.Println("Requested Fetch of dataset: ", datasetId)
		//} else {
		//	fmt.Println("Unable to request download command: ", fetchResponse.Status)
		//	log.Errorf("Unable to request download command: %v", fetchResponse.Status)
		//}
	},
}

func init() {

}
