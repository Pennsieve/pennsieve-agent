/*
Copyright © 2022 University of Pennsylvania <support@pennsieve.io>>

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
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/pennsieve-agent/api"
	"github.com/pennsieve/pennsieve-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var (
	bucket string
	prefix string
)

var UploadCmd = &cobra.Command{
	Use:   "upload [flags] [PATH] [...PATH]",
	Short: "Upload files to the Pennsieve platform.",
	Long:  `Upload files to the Pennsieve platform.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("upload called")

		client := pennsieve.NewClient() // Create simple uninitialized client
		activeUser, err := api.GetActiveUser(client)

		bucket = "pennsieve-dev-test-new-upload"

		apiToken := viper.GetString(activeUser.Profile + ".api_token")
		apiSecret := viper.GetString(activeUser.Profile + ".api_secret")
		client.Authentication.Authenticate(apiToken, apiSecret)

		if err != nil {
			fmt.Println("ERROR")
		}

		client.Authentication.GetAWSCredsForUser()

		paths := args[0]
		uploadToAWS(*client, paths)

	},
}

func init() {
	UploadCmd.Flags().BoolP("recursive", "r",
		false, "Upload folder recursively")
}

type fileWalk chan string

func (f fileWalk) Walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if !info.IsDir() {
		f <- path
	}
	return nil
}

// uploadToAWS implements method to recursively upload path to S3 Bucket
func uploadToAWS(client pennsieve.Client, localPath string) {
	walker := make(fileWalk)
	go func() {
		// Gather the files to upload by walking the path recursively
		if err := filepath.Walk(localPath, walker.Walk); err != nil {
			log.Fatalln("Walk failed:", err)
		}
		close(walker)
	}()

	cfg, err := config.LoadDefaultConfig(context.TODO(), // Hard coded credentials.
		config.WithCredentialsProvider(
			credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     *client.AWSCredentials.AccessKeyId,
					SecretAccessKey: *client.AWSCredentials.SecretKey,
					SessionToken:    *client.AWSCredentials.SessionToken,
					Source:          "example hard coded credentials",
				},
			}))
	if err != nil {
		log.Fatal(err)
	}

	//cfg, err := config.LoadDefaultConfig(config.WithCredentialsProvider())
	if err != nil {
		log.Fatalln("error:", err)
	}

	// For each file found walking, upload it to Amazon S3
	uploader := manager.NewUploader(s3.NewFromConfig(cfg))
	for path := range walker {
		rel, err := filepath.Rel(localPath, path)
		if err != nil {
			log.Fatalln("Unable to get relative path:", path, err)
		}
		file, err := os.Open(path)
		if err != nil {
			log.Println("Failed opening file", path, err)
			continue
		}

		fileInfo, err := file.Stat()

		defer file.Close()

		p := mpb.New()
		reader := &CustomReader{
			fp:      file,
			size:    fileInfo.Size(),
			signMap: map[int64]struct{}{},
			bar: p.AddBar(fileInfo.Size(),
				mpb.PrependDecorators(
					decor.Name("uploading..."),
					decor.Percentage(decor.WCSyncSpace),
				),
			),
		}

		result, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket: &bucket,
			Key:    aws.String(filepath.Join(prefix, rel)),
			Body:   reader,
		})
		if err != nil {
			log.Fatalln("Failed to upload", path, err)
		}
		log.Println("Uploaded", path, result.Location)
	}
}

type CustomReader struct {
	fp      *os.File
	size    int64
	read    int64
	bar     *mpb.Bar
	signMap map[int64]struct{}
	mux     sync.Mutex
}

func (r *CustomReader) Read(p []byte) (int, error) {
	return r.fp.Read(p)
}

func (r *CustomReader) ReadAt(p []byte, off int64) (int, error) {
	n, err := r.fp.ReadAt(p, off)
	if err != nil {
		return n, err
	}

	r.bar.SetTotal(r.size, false)

	r.mux.Lock()
	r.read += int64(n)
	r.bar.SetCurrent(r.read)
	r.mux.Unlock()

	return n, err
}

func (r *CustomReader) Seek(offset int64, whence int) (int64, error) {
	return r.fp.Seek(offset, whence)
}
