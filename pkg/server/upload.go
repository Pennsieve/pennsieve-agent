// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/manifest/manifestFile"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

type record struct {
	sourcePath string
	targetPath string
	targetName string
	uploadId   string
	status     string
}

//type recordWalk chan record
type syncResult chan []manifestFile.FileStatusDTO

// ----------------------------------------------
// UPLOAD FUNCTIONS
// ----------------------------------------------

// CancelUpload cancels an ongoing upload.
func (s *server) CancelUpload(ctx context.Context,
	request *pb.CancelUploadRequest) (*pb.SimpleStatusResponse, error) {

	// TODO: Maybe only cancel uploadSessions that are actively running?

	cancelCount := 0
	s.cancelFncs.Range(func(k, v interface{}) bool {

		session := v.(uploadSession)
		if !request.CancelAll {
			// Only cancel if the manifest id matches requested id
			if session.manifestId == request.ManifestId {
				session.cancelFnc()
				s.sendCancelSubscribers(fmt.Sprintf("Cancelling all uploads."))
				cancelCount += 1
				return false
			}
		} else {
			// Cancel all upload sessions.
			session.cancelFnc()
			s.sendCancelSubscribers(fmt.Sprintf("Cancelling uploading manifest: %d", session.manifestId))
			cancelCount += 1
		}

		return true
	})

	return &pb.SimpleStatusResponse{
		Status: fmt.Sprintf("Succesfully cancelled %d upload sessions", cancelCount)}, nil
}

// UploadManifest uploads all files associated with the provided manifest
func (s *server) UploadManifest(ctx context.Context,
	request *pb.UploadManifestRequest) (*pb.SimpleStatusResponse, error) {

	s.messageSubscribers(fmt.Sprintf("Server starting upload manifest %d.", request.ManifestId))

	var m *store.Manifest
	m, err := s.Manifest.GetManifest(request.ManifestId)
	if err != nil {
		log.Fatalln("Cannot get Manifest based on ID.")
		return nil, err
	}

	s.messageSubscribers("Uploading files to cloud.")

	// On runtime panic, log the stacktrace but keep server alive
	defer func() {
		if x := recover(); x != nil {
			// recovering from a panic; x contains whatever was passed to panic()
			log.Error("Run time panic: %v", x)
			log.Error("Stacktrace: \n %s", string(debug.Stack()))
		}
	}()

	tickerDone := make(chan bool)
	ticker := time.NewTicker(10 * time.Second)

	// Ticker to get status updates from the server periodically
	syncTickerDelay := 15 * time.Minute // time to continue syncing files after upload complete
	go func() {
		for {
			select {
			case <-tickerDone:
				ticker.Stop()
				log.Println("Stopped syncing manifest: ", m.Id)
				return
			case <-ticker.C:

				// Checking status of files on server and verify.
				// This should return a list of files that have recently been finalized and then set the status of
				// those files to "Verified" on the server.
				s.Manifest.VerifyFinalizedStatus(m)

			}
		}
	}()

	// Manager:
	// 1. Create thread to sync
	// 2. Wait for sync to be done
	// 3. Create thread for upload
	go func() {

		// Create Context and store cancel function in server object.
		ctx, cancelFnc := context.WithCancel(context.Background())
		session := uploadSession{
			manifestId: request.GetManifestId(),
			cancelFnc:  cancelFnc,
		}
		s.cancelFncs.Store(request.GetManifestId(), session)

		// Step 1. Synchronize all files
		s.syncProcessor(ctx, m)

		// Step 2. Upload all files
		s.uploadProcessor(ctx, m)

		// Wait X minutes before cancelling sync thread.
		// TODO: improve by registering sync thread so we never have more than 1 sync thread per manifest.
		syncTimer := time.NewTimer(syncTickerDelay)
		<-syncTimer.C
		tickerDone <- true

	}()

	response := pb.SimpleStatusResponse{Status: "Upload initiated."}
	return &response, nil
}

func (s *server) uploadProcessor(ctx context.Context, m *store.Manifest) {

	nrWorkers := viper.GetInt("agent.upload_workers")
	walker := make(chan store.ManifestFile, nrWorkers)
	results := make(chan int, nrWorkers)

	var uploadWg sync.WaitGroup

	// Database crawler to fetch rows
	go func() {

		defer close(walker)

		requestStatus := []manifestFile.Status{
			manifestFile.Registered,
		}

		s.Manifest.ManifestFilesToChannel(ctx, m.Id, requestStatus, walker)

	}()

	// Upload Handler
	go func() {

		chunkSize := viper.GetInt64("agent.upload_chunk_size")
		nrWorkers := viper.GetInt("agent.upload_workers")

		client := s.client
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion("us-east-1"),
			config.WithCredentialsProvider(
				pennsieve.AWSCredentialProviderWithExpiration{
					AuthService: client.Authentication,
				},
			),
		)

		if err != nil {
			log.Fatal(err)
		}

		// For each file found walking, upload it to Amazon S3
		ctx, cancelFnc := context.WithCancel(context.Background())
		session := uploadSession{
			manifestId: m.Id,
			cancelFnc:  cancelFnc,
		}
		s.cancelFncs.Store(m.Id, session)
		defer cancelFnc()

		// Create an S3 Client with the config
		s3Client := s3.NewFromConfig(cfg)

		// Create an uploader with the client and custom options
		uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
			u.PartSize = chunkSize * 1024 * 1024 // ...MB per part
		})

		s.updateSubscribers(0, 0, "", 0, pb.SubscribeResponse_UploadResponse_INIT)

		// Initiate the upload workers
		for w := 1; w <= nrWorkers; w++ {
			uploadWg.Add(1)
			log.Println("Starting upload worker: ", w)
			w := int32(w)
			go func() {
				defer func() {
					log.Println("Closing upload worker: ", w)
					uploadWg.Done()
				}()

				err := s.uploadWorker(ctx, w, walker, results, m.NodeId.String, uploader, cfg, client.UploadBucket)
				if err != nil {
					log.Println("Error in Upload Worker:", err)
				}
			}()
		}

		uploadWg.Wait()
		log.Println("Upload Completed")
		s.updateSubscribers(0, 0, "", 0, pb.SubscribeResponse_UploadResponse_COMPLETE)

		log.Println("Returned from uploader for manifest: ", m.Id)
	}()

}

func (s *server) uploadWorker(ctx context.Context, workerId int32,
	jobs <-chan store.ManifestFile, results chan<- int, manifestNodeId string,
	uploader *manager.Uploader, cfg aws.Config, uploadBucket string) error {

	for record := range jobs {

		file, err := os.Open(record.SourcePath)
		if err != nil {
			log.Println("Failed opening file", record.SourcePath, err)
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

		s3Key := aws.String(fmt.Sprintf("%s/%s", manifestNodeId, record.UploadId))

		_, err = uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(uploadBucket),
			Key:    s3Key,
			Body:   reader,
		})
		if err != nil {

			// If Cancelled, need to manually abort upload on S3 to remove partial upload on S3. For other errors, this
			// is done automatically by the manager.
			if errors.Is(err, context.Canceled) {

				s.messageSubscribers(fmt.Sprintf("Upload canceled."))

				var mu manager.MultiUploadFailure
				if errors.As(err, &mu) {
					//log.Println("Cancelling multi-part upload: ", record.sourcePath)

					s3Session := s3.NewFromConfig(cfg)
					input := &s3.AbortMultipartUploadInput{
						Bucket:   aws.String(uploadBucket),
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
						Bucket:   aws.String(uploadBucket),
						Key:      aws.String(*s3Key),
						UploadId: aws.String(mu.UploadID()),
					}

					maxRetry := 10
					iter := 0
					for {
						_, err = s3Session.ListParts(context.Background(), inputListParts)
						iter += 1
						if err != nil {
							log.Println("Multi-part upload cancelled: ", record.SourcePath)
							break
						} else if iter == maxRetry {
							log.Println("Maximum retries for cancelling multipart upload: ", record.SourcePath)
							break
						} else {
							time.Sleep(500 * time.Millisecond)
						}
					}
				}

				err = file.Close()
				if err != nil {
					log.Fatalln("Could not close file.")
				}

				break
			} else {
				// Process error generically
				log.Println("Failed to upload", record.SourcePath)
				log.Println("Error:", err.Error())

				s.messageSubscribers(fmt.Sprintf("Upload Failed: see log for details."))

				err = file.Close()
				if err != nil {
					log.Fatalln("Could not close file.")
				}
			}

			continue

		}

		err = file.Close()
		if err != nil {
			log.Fatalln("Could not close file.")
		}

		err = s.Manifest.SetFileStatus(record.UploadId.String(), manifestFile.Uploaded)
		if err != nil {
			log.Fatalln("Could not update status of file. Here is why: ", err)
		}
	}
	return nil
}

// ----------------------------------------------
// CUSTOM READER FUNCTIONS
// ----------------------------------------------

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
	r.s.updateSubscribers(r.size, r.read, r.fp.Name(), r.workerId, pb.SubscribeResponse_UploadResponse_IN_PROGRESS)

	return n, err
}

func (r *CustomReader) Seek(offset int64, whence int) (int64, error) {
	return r.fp.Seek(offset, whence)
}

// ----------------------------------------------
// SUBSCRIBER UPDATES
// ----------------------------------------------

// updateSubscribers sends upload-progress updates to all grpc-update subscribers.
func (s *server) updateSubscribers(total int64, current int64, name string, workerId int32,
	status pb.SubscribeResponse_UploadResponse_UploadStatus) {
	// A list of clients to unsubscribe in case of error
	var unsubscribe []int32

	// Iterate over all subscribers and send data to each client
	s.subscribers.Range(func(k, v interface{}) bool {
		id, ok := k.(int32)
		if !ok {
			log.Error("Failed to cast subscriber key: %T", k)
			return false
		}
		sub, ok := v.(sub)
		if !ok {
			log.Error("Failed to cast subscriber value: %T", v)
			return false
		}
		// Send data over the gRPC stream to the client
		if err := sub.stream.Send(&pb.SubscribeResponse{
			Type: 1,
			MessageData: &pb.SubscribeResponse_UploadStatus{
				UploadStatus: &pb.SubscribeResponse_UploadResponse{
					FileId:   name,
					Total:    total,
					Current:  current,
					WorkerId: workerId,
					Status:   status,
				}},
		}); err != nil {

			select {
			case sub.finished <- true:
				log.Info("Unsubscribed client: %d", id)
			default:
				log.Warn("Failed to send data to client: %v", err)
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

// sendCancelSubscribers Send Cancel Message to Subscribers
func (s *server) sendCancelSubscribers(message string) {
	// A list of clients to unsubscribe in case of error
	var unsubscribe []int32

	// Iterate over all subscribers and send data to each client
	s.subscribers.Range(func(k, v interface{}) bool {
		id, ok := k.(int32)
		if !ok {
			log.Error("Failed to cast subscriber key: %T", k)
			return false
		}
		sub, ok := v.(sub)
		if !ok {
			log.Error("Failed to cast subscriber value: %T", v)
			return false
		}
		// Send data over the gRPC stream to the client
		if err := sub.stream.Send(&pb.SubscribeResponse{
			Type: pb.SubscribeResponse_UPLOAD_CANCEL,
			MessageData: &pb.SubscribeResponse_EventInfo{
				EventInfo: &pb.SubscribeResponse_EventResponse{Details: message}},
		}); err != nil {
			select {
			case sub.finished <- true:
				log.Info("Unsubscribed client: %d", id)
			default:
				log.Warn("Failed to send data to client: %v", err)
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
