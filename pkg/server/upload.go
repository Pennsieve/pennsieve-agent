// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"database/sql"
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
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"
)

type record struct {
	sourcePath string
	targetPath string
	s3Key      string
}

type recordWalk chan record

var uploadWg sync.WaitGroup

// API ENDPOINT IMPLEMENTATIONS
// --------------------------------------------

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

	// TODO: create second channel to update upload status
	chunkSize := viper.GetInt64("agent.upload_chunk_size")
	nrWorkers := viper.GetInt("agent.upload_workers")

	walker := make(recordWalk, nrWorkers)
	results := make(chan int, nrWorkers)

	// Database crawler: the database crawler populates a channel with records to be uploaded
	go func() {

		// Close walker when all records for manifest were added to channel
		defer func() {
			close(walker)
		}()

		rows, err := dbconfig.DB.Query(
			"SELECT source_path, target_path, s3_key FROM upload_record WHERE session_id=?", request.ManifestId)
		if err != nil {
			log.Fatal(err)
		}
		defer func(rows *sql.Rows) {
			err := rows.Close()
			if err != nil {
				log.Println("Unable to close rows in Upload.")
			}
		}(rows)

		// Iterate over rows for manifest and add row to channel to be picked up by worker.
		for rows.Next() {
			r := record{}
			err := rows.Scan(&r.sourcePath, &r.targetPath, &r.s3Key)
			if err != nil {
				log.Fatal(err)
			}
			walker <- r
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}

	}()

	// Upload Manager: the upload manager creates n upload workers to upload files provided by the Database Crawler.
	go func() {
		cfg, err := config.LoadDefaultConfig(context.TODO(),
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
		session := uploadSession{
			manifestId: request.GetManifestId(),
			cancelFnc:  cancelFnc,
		}
		s.cancelFncs.Store(request.GetManifestId(), session)
		defer cancelFnc()

		// Create an S3 Client with the config
		s3Client := s3.NewFromConfig(cfg)

		// Create an uploader with the client and custom options
		uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
			u.PartSize = chunkSize * 1024 * 1024 // 64MB per part
		})

		log.Println("Uploader: ", uploader.PartSize)

		// Initiate the upload workers
		for w := 1; w <= nrWorkers; w++ {
			uploadWg.Add(1)
			log.Println("starting worker:", w)
			w := int32(w)
			go func() {
				err := s.uploadWorker(ctx, w, walker, results, request.ManifestId, uploader, cfg)
				if err != nil {
					log.Println("Error in Upload Worker:", err)
				}
			}()
		}

		// Wait until all workers and record crawler
		uploadWg.Wait()

		log.Println("Returned from uploader")
	}()

	response := pb.SimpleStatusResponse{Status: "Upload initiated."}
	return &response, nil
}

// HELPER FUNCTIONS
// ----------------------------------------------

func (s *server) uploadWorker(ctx context.Context, workerId int32,
	jobs <-chan record, results chan<- int, manifestId string, uploader *manager.Uploader, cfg aws.Config) error {

	defer func() {
		log.Println("Closing Worker: ", workerId)
		uploadWg.Done()
	}()

	for record := range jobs {

		file, err := os.Open(record.sourcePath)
		if err != nil {
			log.Println("Failed opening file", record.sourcePath, err)
			continue
		}

		fileInfo, err := file.Stat()

		reader := &CustomReader{
			workerId: workerId,
			fp:       file,
			size:     fileInfo.Size(),
			signMap:  map[int64]struct{}{},
			s:        s,
		}

		s3Key := aws.String(filepath.Join(manifestId, record.targetPath))

		_, err = uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: &s.uploadBucket,
			Key:    s3Key,
			Body:   reader,
		})
		if err != nil {
			//s.messageSubscribers(err.Error())

			// If Cancelled, need to manually abort upload on S3 to remove partial upload on S3. For other errors, this
			// is done automatically by the manager.
			if errors.Is(err, context.Canceled) {
				var mu manager.MultiUploadFailure
				if errors.As(err, &mu) {
					//log.Println("Cancelling multi-part upload: ", record.sourcePath)

					s3Session := s3.NewFromConfig(cfg)
					input := &s3.AbortMultipartUploadInput{
						Bucket:   aws.String(s.uploadBucket),
						Key:      aws.String(*s3Key),
						UploadId: aws.String(mu.UploadID()),
					}

					_, err := s3Session.AbortMultipartUpload(context.Background(), input)
					if err != nil {
						log.Println("Failed to abort multipart after cancelling: ", err)
						return err
					}

					// Try to get the parts of the removed multipart upload. This should fail as all parts are removed
					// but can theoretically succeed if we delete parts at the same time that they are being written.
					// In that case, we try again to delete the multipart upload.
					inputListParts := &s3.ListPartsInput{
						Bucket:   aws.String(s.uploadBucket),
						Key:      aws.String(*s3Key),
						UploadId: aws.String(mu.UploadID()),
					}

					maxRetry := 10
					iter := 0
					for {
						_, err = s3Session.ListParts(context.Background(), inputListParts)
						iter += 1
						if err != nil {
							log.Println("Multi-part upload cancelled: ", record.sourcePath)
							break
						} else if iter == maxRetry {
							log.Println("Maximum retries for cancelling multipart upload: ", record.sourcePath)
							break
						} else {
							time.Sleep(500 * time.Millisecond)
						}
					}
				}
				break
			} else {
				// Process error generically
				log.Println("Failed to upload", record.sourcePath)
				log.Println("Error:", err.Error())
			}

			continue

		}

		err = file.Close()
		if err != nil {
			log.Fatalln("Could not close file.")
		}
	}
	return nil
}

type CustomReader struct {
	workerId int32
	fp       *os.File
	size     int64
	read     int64
	signMap  map[int64]struct{}
	s        *server
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
	r.s.updateSubscribers(r.size, r.read, r.fp.Name(), r.workerId)

	return n, err
}

func (r *CustomReader) Seek(offset int64, whence int) (int64, error) {
	return r.fp.Seek(offset, whence)
}

// updateSubscribers sends upload-progress updates to all grpc-update subscribers.
func (s *server) updateSubscribers(total int64, current int64, name string, workerId int32) {
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
					FileId:   name,
					Total:    total,
					Current:  current,
					WorkerId: workerId,
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
