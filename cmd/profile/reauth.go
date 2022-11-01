package profile

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/pennsieve/pennsieve-agent/cmd/whoami"
	"github.com/pennsieve/pennsieve-agent/pkg/config"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ReauthCmd = &cobra.Command{
	Use:   "reauth",
	Short: "Displays information about the logged in user.",
	Long:  `Displays information about the logged in user.`,
	Run: func(cmd *cobra.Command, args []string) {

		req := v1.ReAuthenticateRequest{}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server. See log for details.")
			log.Error("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := v1.NewAgentClient(conn)
		userResponse, err := client.ReAuthenticate(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to complete getUser command: %v", err))
			return
		}

		db, _ := config.InitializeDB()
		userSettingsStore := store.NewUserSettingsStore(db)
		userInfoStore := store.NewUserInfoStore(db)
		_, err = config.InitPennsieveClient(userSettingsStore, userInfoStore)
		if err != nil {
			log.Fatalln("Cannot connect to Pennsieve.")
		}

		showFull, _ := cmd.Flags().GetBool("full")
		whoami.PrettyPrint(userResponse, showFull)
	},
}

func init() {
}
