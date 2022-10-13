package version

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var Version = "development"

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows the version of the Agent and CLI.",
	Run: func(cmd *cobra.Command, args []string) {

		req := v1.VersionRequest{}
		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := v1.NewAgentClient(conn)

		versionResponse, err := client.Version(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete Version command: %v", err))
			return
		}

		fmt.Println("Pennsieve Agent")
		fmt.Println(fmt.Sprintf("Agent Version  :%20s\n"+
			"CLI Version    :%20s\n"+
			"Log Level      :%20s\n", versionResponse.Version, Version, versionResponse.LogLevel))

	},
}
