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
package whoami

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/pennsieve/pennsieve-agent/pkg/config"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
)

// whoamiCmd represents the whoami command
var WhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Displays information about the logged in user.",
	Long:  `Displays information about the logged in user.`,
	Run: func(cmd *cobra.Command, args []string) {

		req := v1.GetUserRequest{}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := v1.NewAgentClient(conn)

		userResponse, err := client.GetUser(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete getUser command: %v", err))
			return
		}

		db, _ := config.InitializeDB()
		userSettingsStore := store.NewUserSettingsStore(db)
		userInfoStore := store.NewUserInfoStore(db)
		pennsieveClient, err := config.InitPennsieveClient(userSettingsStore, userInfoStore)
		if err != nil {
			log.Fatalln("Cannot connect to Pennsieve.")
		}

		showFull, _ := cmd.Flags().GetBool("full")
		PrettyPrint(userResponse, pennsieveClient.Authentication.BaseUrl, showFull)
	},
}

func init() {
	WhoamiCmd.Flags().BoolP("full", "f",
		false, "Show expanded information")
}

// PrettyPrint renders a table with current userinfo to terminal
func PrettyPrint(info *v1.UserResponse, host string, showFull bool) {
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
			{"SESSION-TOKEN", info.SessionToken},
			{"HOST", host},
		})
	}

	t.Render()
}
