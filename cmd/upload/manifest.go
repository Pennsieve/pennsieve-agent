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
	"github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	"github.com/pennsieve/pennsieve-agent/pkg/subscriber"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"math/rand"
	"strconv"
	"time"
)

var ManifestCmd = &cobra.Command{
	Use:   "manifest <manifestId>",
	Short: "Upload files to the Pennsieve platform.",
	Long:  `Upload files to the Pennsieve platform.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// Check input argument (needs to be an integer)
		i, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			fmt.Printf("Error: <manifestId> should be an integer.")
			return
		}

		manifestId := int32(i)
		req := v1.UploadManifestRequest{ManifestId: manifestId}
		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := v1.NewAgentClient(conn)
		_, err = client.UploadManifest(context.Background(), &req)
		if err != nil {
			shared.HandleAgentError(err, fmt.Sprintln("Error uploading manifest: ", err))
		}

		fmt.Println(fmt.Sprintf("\nUpload initiated for manifest: %d.\n You can safely Ctr-C as uploading process will continue to run in the background."+
			"\n\n  Use \"pennsieve agent subscribe\" to track progress of the uploaded files.\n\n"+
			"  Use \"pennsieve upload cancel %d\" to cancel the current upload session.", manifestId, manifestId))

		fmt.Println("\n------------")

		// Subscribe to messages from the GRPC server and quit when upload is complete.
		r1 := rand.New(rand.NewSource(time.Now().UnixNano()))
		SubscribeClient, err := subscriber.NewSubscriberClient(int32(r1.Intn(100)))
		if err != nil {
			fmt.Println("Unable to track uploads. Please check logs to verify files are uploaded.")
		}
		SubscribeClient.Start([]v1.SubscribeResponse_MessageType{
			v1.SubscribeResponse_UPLOAD_STATUS, v1.SubscribeResponse_EVENT}, subscriber.StopOnStatus{
			Enable: true,
			OnType: []v1.SubscribeResponse_MessageType{v1.SubscribeResponse_UPLOAD_STATUS},
		})
	},
}
