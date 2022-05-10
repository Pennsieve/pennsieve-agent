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
	Use:   "manifest <manifestId>",
	Short: "Upload files to the Pennsieve platform.",
	Long:  `Upload files to the Pennsieve platform.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		manifestId := args[0]
		req := pb.UploadManifestRequest{
			ManifestId: manifestId,
		}

		port := viper.GetString("agent.port")

		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
		}
		defer conn.Close()

		client := pb.NewAgentClient(conn)
		_, err = client.UploadManifest(context.Background(), &req)
		if err != nil {
			fmt.Println("Error uploading file: ", err)
		}

		fmt.Println(fmt.Sprintf("\nUpload initiated for manifest: %s\n\nUse "+
			"\"pennsieve-agent agent subscribe\" to track progress of the uploaded files.\n\n"+
			"Use \"pennsieve-agent upload cancel %s\" to cancel the current upload session.", manifestId, manifestId))
	},
}

func init() {

}
