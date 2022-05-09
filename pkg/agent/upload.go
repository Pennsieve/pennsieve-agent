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
	dbconfig "github.com/pennsieve/pennsieve-agent/pkg/db"
	pb "github.com/pennsieve/pennsieve-agent/protos"
	"github.com/pkg/errors"
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

// CancelUpload cancels an ongoing upload.
func (s *server) CancelUpload(ctx context.Context, request *pb.CancelRequest) (*pb.SimpleStatusResponse, error) {

	// TODO: Maybe only cancel uploadSessions that are actively running?

	cancelCount := 0
	s.cancelFncs.Range(func(k, v interface{}) bool {

		session := v.(uploadSession)
		if !request.CancelAll {
			// Only cancel if the manifest id matches requested id
			if session.manifestId == request.ManifestId {
				session.cancelFnc()
				s.sendCancelSubscribers(session.manifestId)
				cancelCount += 1
				return false
			}
		} else {
			// Cancel all upload sessions.
			session.cancelFnc()
			s.sendCancelSubscribers(session.manifestId)
			cancelCount += 1
		}

		return true
	})

	return &pb.SimpleStatusResponse{
		Status: fmt.Sprintf("Succesfully cancelled %d upload sessions", cancelCount)}, nil
}

// UploadManifest uploads all files associated with the provided manifest
func (s *server) UploadManifest(ctx context.Context, request *pb.UploadManifestRequest) (*pb.SimpleStatusResponse, error) {

	// On runtime panic, log the stacktrace but keep server alive
	defer func() {
		if x := recover(); x != nil {
			// recovering from a panic; x contains whatever was passed to panic()
			log.Printf("Run time panic: %v", x)
			log.Printf("Stacktrace: \n %s", string(debug.Stack()))
		}
	}()

	client := api.PennsieveClient
	client.Authentication.GetAWSCredsForUser()

	// TODO: Check if this is causing a leak when cancelling upload
	// TODO: create second channel to update upload status
	walker := make(fileWalk)
	go func() {
		var (
			sourcePath string
			s3Key      string
		)

		rows, err := dbconfig.DB.Query(
			"SELECT source_path, s3_key FROM upload_record WHERE session_id=?", request.ManifestId)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		for rows.Next() {
			err := rows.Scan(&sourcePath, &s3Key)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Adding ", sourcePath)
			walker <- sourcePath
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}

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
	//s.cancelFnc = cancelFnc
	session := uploadSession{
		manifestId: request.GetManifestId(),
		cancelFnc:  cancelFnc,
	}
	s.cancelFncs.Store(request.GetManifestId(), session)
	defer cancelFnc()

	uploader := manager.NewUploader(s3.NewFromConfig(cfg))
	for path := range walker {

		rel, err := filepath.Rel("/Users/joostw/", path)
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
			Bucket: &s.uploadBucket,
			Key:    s3Key,
			Body:   reader,
		})
		if err != nil {
			s.messageSubscribers(err.Error())
			log.Println("Failed to upload", path, err)

			// If Cancelled, need to manually abort upload on S3 to remove partial upload on S3. For other errors, this
			// is done automatically by the manager.
			if errors.Is(err, context.Canceled) {
				var mu manager.MultiUploadFailure
				if errors.As(err, &mu) {
					// Process error and its associated uploadID

					s3Session := s3.NewFromConfig(cfg)

					input := &s3.AbortMultipartUploadInput{
						Bucket:   aws.String(s.uploadBucket),
						Key:      aws.String(*s3Key),
						UploadId: aws.String(mu.UploadID()),
					}

					_, err := s3Session.AbortMultipartUpload(context.Background(), input)
					if err != nil {
						log.Println("Failed to abort multipart after cancelling: ", err)
						response := pb.SimpleStatusResponse{Status: "Upload failed."}
						return &response, err
					}

					inputListParts := &s3.ListPartsInput{
						Bucket:   aws.String(s.uploadBucket),
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

	log.Println("Returned from uploader")
	response := pb.SimpleStatusResponse{Status: "Upload completed."}
	return &response, nil
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
	r.s.updateSubscribers(r.size, r.read, r.fp.Name())

	return n, err
}

func (r *CustomReader) Seek(offset int64, whence int) (int64, error) {
	return r.fp.Seek(offset, whence)
}

// updateSubscribers sends upload-progress updates to all grpc-update subscribers.
func (s *server) updateSubscribers(total int64, current int64, name string) {
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

// Send Cancel Message to Subscribers
func (s *server) sendCancelSubscribers(manifestId string) {
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
			Type: pb.SubsrcribeResponse_UPLOAD_CANCEL,
			MessageData: &pb.SubsrcribeResponse_EventInfo{
				EventInfo: &pb.SubsrcribeResponse_EventResponse{Details: manifestId}},
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
