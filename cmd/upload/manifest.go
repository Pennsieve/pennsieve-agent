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

var ManifestCmd = &cobra.Command{
	Use:   "manifest [flags] [PATH] [...PATH]",
	Short: "Upload files to the Pennsieve platform.",
	Long:  `Upload files to the Pennsieve platform.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("upload called")

		req := pb.UploadManifestRequest{
			ManifestId: args[0],
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		uploadResponse, err := client.UploadManifest(context.Background(), &req)
		if err != nil {
			fmt.Println("Error uploading file: ", err)
		}
		fmt.Println(uploadResponse)
	},
}

func init() {

}
