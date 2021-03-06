/*
Copyright © 2022 University of Pennsylvania <support@server>>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package profile

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"os"
)

var SwitchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch between user profiles.",
	Long:  `Switch between user profiles that are defined in the Pennsieve Config file.`,

	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		selectedProfile := args[0]

		req := pb.SwitchProfileRequest{
			Profile: selectedProfile,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		switchResponse, err := client.SwitchProfile(context.Background(), &req)
		if err != nil {
			st := status.Convert(err)
			fmt.Println(st.Message())
			return
		}

		prettyPrint(*switchResponse, false)

	},
}

func init() {
}

// prettyPrint renders a table with current userinfo to terminal
func prettyPrint(info pb.UserResponse, showFull bool) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendRows([]table.Row{
		{"NAME", info.Name},
		{"USER ID", info.Id},
		{"ORGANIZATION", info.OrganizationName},
		{"ORGANIZATION ID", info.OrganizationId},
	})
	if showFull {
		t.AppendRows([]table.Row{
			{"PROFILE", info.Profile},
			{"ENVIRONMENT", info.Environment},
		})
	}

	t.Render()
}
