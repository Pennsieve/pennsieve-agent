package agent

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/pennsieve-agent/pkg/api"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/pennsieve/pennsieve-go"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
)

var (
	bucket string
	prefix string
)

type fileWalk chan string

func (f fileWalk) Walk(path string, info fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !info.IsDir() {
		f <- path
	}
	return nil
}

// uploadToAWS implements method to recursively upload path to S3 Bucket
func (s *server) uploadToAWS(client pennsieve.Client, localPath string) error {

	bucket = "pennsieve-dev-test-new-upload"

	walker := make(fileWalk)
	go func() {
		// Gather the files to upload by walking the path recursively
		if err := filepath.WalkDir(localPath, walker.Walk); err != nil {
			log.Println("Walk failed:", err)
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
					Source:          "Pennsieve Agent",
				},
			}))
	if err != nil {
		log.Fatal(err)
	}

	// For each file found walking, upload it to Amazon S3
	ctx, cancelFnc := context.WithCancel(context.Background())
	s.cancelFnc = cancelFnc
	defer cancelFnc()

	uploader := manager.NewUploader(s3.NewFromConfig(cfg))
	for path := range walker {

		rel, err := filepath.Rel(localPath, path)
		if err != nil {
			log.Println("Unable to get relative path:", path, err)
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			log.Println("Failed opening file", path, err)
			continue
		}
		defer file.Close()
		fileInfo, err := file.Stat()

		reader := &CustomReader{
			fp:      file,
			size:    fileInfo.Size(),
			signMap: map[int64]struct{}{},
			s:       s,
		}

		s3Key := aws.String(filepath.Join(prefix, rel))

		_, err = uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: &bucket,
			Key:    s3Key,
			Body:   reader,
		})
		if err != nil {
			messageSubscribers(s, err.Error())
			log.Println("Failed to upload", path, err)

			// If Cancelled, need to manually abort upload on S3 to remove partial upload on S3. For other errors, this
			// is done automatically by the manager.
			if errors.Is(err, context.Canceled) {
				var mu manager.MultiUploadFailure
				if errors.As(err, &mu) {
					// Process error and its associated uploadID

					s3Session := s3.NewFromConfig(cfg)

					input := &s3.AbortMultipartUploadInput{
						Bucket:   aws.String(bucket),
						Key:      aws.String(*s3Key),
						UploadId: aws.String(mu.UploadID()),
					}

					_, err := s3Session.AbortMultipartUpload(context.Background(), input)
					if err != nil {
						log.Println("Failed to abort multipart after cancelling: ", err)
						return err
					}

					inputListParts := &s3.ListPartsInput{
						Bucket:   aws.String(bucket),
						Key:      aws.String(*s3Key),
						UploadId: aws.String(mu.UploadID()),
					}

					listPartsOutput, err := s3Session.ListParts(context.Background(), inputListParts)
					if err != nil {
						log.Println("ListPartError:", err)
					}
					log.Println("ListPartResponse: ", listPartsOutput)

				} else {
					// Process error generically
					log.Println("Error:", err.Error())
				}
				break
			}

			continue

		}
	}

	return nil
}

// UploadPath recursively uploads a folder to the Pennsieve Platform.
func (s *server) UploadPath(ctx context.Context, request *pb.UploadRequest) (*pb.UploadResponse, error) {

	// On runtime panic, log the stacktrace but keep server alive
	defer func() {
		if x := recover(); x != nil {
			// recovering from a panic; x contains whatever was passed to panic()
			log.Printf("Run time panic: %v", x)
			log.Printf("Stacktrace: \n %s", string(debug.Stack()))
		}
	}()

	client := api.PennsieveClient
	activeUser, err := api.GetActiveUser()
	if err != nil {
		fmt.Println(err)

	}

	apiToken := viper.GetString(activeUser.Profile + ".api_token")
	apiSecret := viper.GetString(activeUser.Profile + ".api_secret")
	client.Authentication.Authenticate(apiToken, apiSecret)

	if err != nil {
		fmt.Println("ERROR")
	}

	client.Authentication.GetAWSCredsForUser()

	err = s.uploadToAWS(*client, request.BasePath)

	log.Println("Returned from uploader")
	response := pb.UploadResponse{Status: "Upload completed."}
	return &response, nil
}

// CancelUpload cancels an ongoing upload.
func (s *server) CancelUpload(ctx context.Context, request *pb.CancelRequest) (*pb.CancelResponse, error) {
	s.cancelFnc()
	return &pb.CancelResponse{
		Status: "Success"}, nil
}

type CustomReader struct {
	fp      *os.File
	size    int64
	read    int64
	signMap map[int64]struct{}
	s       *server
}

func (r *CustomReader) Read(p []byte) (int, error) {
	return r.fp.Read(p)
}

func (r *CustomReader) ReadAt(p []byte, off int64) (int, error) {
	n, err := r.fp.ReadAt(p, off)
	if err != nil {
		return n, err
	}

	r.read += int64(n)
	updateSubscribers(r.s, r.size, r.read, r.fp.Name())

	return n, err
}

func (r *CustomReader) Seek(offset int64, whence int) (int64, error) {
	return r.fp.Seek(offset, whence)
}

// updateSubscribers sends upload-progress updates to all grpc-update subscribers.
func updateSubscribers(s *server, total int64, current int64, name string) {
	// A list of clients to unsubscribe in case of error
	var unsubscribe []int32

	// Iterate over all subscribers and send data to each client
	s.subscribers.Range(func(k, v interface{}) bool {
		id, ok := k.(int32)
		if !ok {
			log.Printf("Failed to cast subscriber key: %T", k)
			return false
		}
		sub, ok := v.(sub)
		if !ok {
			log.Printf("Failed to cast subscriber value: %T", v)
			return false
		}
		// Send data over the gRPC stream to the client
		if err := sub.stream.Send(&pb.SubsrcribeResponse{
			Type: 1,
			MessageData: &pb.SubsrcribeResponse_UploadStatus{
				UploadStatus: &pb.SubsrcribeResponse_UploadResponse{
					FileId:  name,
					Total:   total,
					Current: current,
				}},
		}); err != nil {
			log.Printf("Failed to send data to client: %v", err)
			select {
			case sub.finished <- true:
				log.Printf("Unsubscribed client: %d", id)
			default:
				// Default case is to avoid blocking in case client has already unsubscribed
			}
			// In case of error the client would re-subscribe so close the subscriber stream
			unsubscribe = append(unsubscribe, id)
		}
		return true
	})

	// Unsubscribe erroneous client streams
	for _, id := range unsubscribe {
		s.subscribers.Delete(id)
	}
}
