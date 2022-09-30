package agent

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var port int32
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Agent",
	Long:  `Stops the Pennsieve agent if it is running in the background.`,
	Run: func(cmd *cobra.Command, args []string) {

		port, _ := cmd.Flags().GetString("port")
		if len(port) == 0 {
			port = viper.GetString("agent.port")
		}

		req := pb.StopRequest{}

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)

		// Check if Pennsieve Server is running at the selected port
		_, err = client.Ping(context.Background(), &pb.PingRequest{})
		if err != nil {
			st := status.Convert(err)
			switch st.Code() {
			case codes.Unavailable:
				fmt.Println("No Pennsieve Agent running on port: ", port)
				return
			default:
				shared.HandleAgentError(err, "Unknown error while stopping Pennsieve Agent Server.")
			}
		}

		// Close the server on that port
		resp, err := client.Stop(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			switch st.Code() {
			case codes.Unavailable:
				fmt.Println("Pennsieve Agent successfully stopped.")
			default:
				shared.HandleAgentError(err, "Unknown error while stopping Pennsieve Agent Server.")
			}
		}

		if resp.Success {
			fmt.Println("Pennsieve Agent successfully stopped.")
		}

	},
}

func init() {
	stopCmd.Flags().StringP("port", "p", "", "Agent Port")
}
