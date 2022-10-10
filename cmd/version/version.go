package version

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows the version of the Agent and CLI.",
	Run: func(cmd *cobra.Command, args []string) {
		req := pb.VersionRequest{}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)

		versionResponse, err := client.Version(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete getUser command: %v", err))
			return
		}

		fmt.Println("Pennsieve Agent")
		fmt.Println(fmt.Sprintf("Version  :%20s\nLog Level:%20s\n", versionResponse.Version, versionResponse.LogLevel))

	},
}
