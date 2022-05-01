package upload

import (
	"context"
	"fmt"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var CancelCmd = &cobra.Command{
	Use:   "cancel [flags] [PATH] [...PATH]",
	Short: "Cancel upload session.",
	Long:  `Cancel upload session.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cancel called")

		req := pb.CancelRequest{}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		uploadResponse, err := client.CancelUpload(context.Background(), &req)
		if err != nil {
			fmt.Println("Error uploading file: ", err)
		}
		fmt.Println(uploadResponse)

	},
}

func init() {

}
