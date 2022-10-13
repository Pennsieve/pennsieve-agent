package profile

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/pennsieve/pennsieve-agent/cmd/whoami"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

var ReauthCmd = &cobra.Command{
	Use:   "reauth",
	Short: "Displays information about the logged in user.",
	Long:  `Displays information about the logged in user.`,
	Run: func(cmd *cobra.Command, args []string) {

		req := pb.ReAuthenticateRequest{}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		userResponse, err := client.ReAuthenticate(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete getUser command: %v", err))
			return
		}

		db, _ := db.InitializeDB()
		userSettingsStore := store.NewUserSettingsStore(db)
		userInfoStore := store.NewUserInfoStore(db)
		pennsieveClient, err := api.InitPennsieveClient(userSettingsStore, userInfoStore)
		if err != nil {
			log.Fatalln("Cannot connect to Pennsieve.")
		}

		showFull, _ := cmd.Flags().GetBool("full")
		whoami.PrettyPrint(userResponse, pennsieveClient.Authentication.BaseUrl, showFull)
	},
}

func init() {
}
