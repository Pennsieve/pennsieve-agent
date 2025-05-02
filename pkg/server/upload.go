// package server implements a gRPC server that runs locally on the clients' computer.
// It implements the endpoints defined in the agent.proto file.

package server

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pennsieve/pennsieve-agent/pkg/models"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/manifest/manifestFile"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	log "github.com/sirupsen/logrus"
)

// ----------------------------------------------
// UPLOAD FUNCTIONS
// ----------------------------------------------

// CancelUpload cancels an ongoing upload.
// TODO: Maybe only cancel uploadSessions that are actively running?
func (s *agentServer) CancelUpload(
	ctx context.Context,
	request *pb.CancelUploadRequest,
) (*pb.SimpleStatusResponse, error) {
	cancelCount := 0
	s.cancelFncs.Range(func(k, v any) bool {
		session := v.(uploadSession)
		if !request.CancelAll { // only cancel if the manifest id matches requested id
			if session.manifestId == request.ManifestId {
				session.cancelFnc()
				s.sendCancelSubscribers("Cancelling all uploads.")
				cancelCount += 1
				return false
			}
		} else { // cancel all upload sessions
			session.cancelFnc()
			s.sendCancelSubscribers(fmt.Sprintf("Cancelling uploading manifest: %d", session.manifestId))
			cancelCount += 1
		}

		return true
	})

	return &pb.SimpleStatusResponse{
		Status: fmt.Sprintf("Succesfully cancelled %d upload sessions", cancelCount),
	}, nil
}

// UploadManifest uploads all files associated with the provided manifest
func (s *agentServer) UploadManifest(
	ctx context.Context,
	request *pb.UploadManifestRequest,
) (*pb.SimpleStatusResponse, error) {
	s.messageSubscribers(fmt.Sprintf("Server starting upload manifest %d.", request.ManifestId))

	manifest, err := s.ManifestService().GetManifest(request.ManifestId)
	if err != nil {
		log.Error("Cannot get Manifest based on ID.")
		return nil, err
	}

	s.messageSubscribers("Uploading files to cloud.")

	// On runtime panic, log the stacktrace but keep server alive
	defer func() {
		if x := recover(); x != nil {
			// recovering from a panic; x contains whatever was passed to panic()
			log.Error(fmt.Sprintf("Run time panic: %v", x))
			log.Error(fmt.Sprintf("Stacktrace: \n %s", string(debug.Stack())))
		}
	}()

	// collect all file status updates in a single buffered channel to serialize writes
	statusUpdates := make(chan models.UploadStatusUpdateMessage, 100)
	go s.startStatusUpdateBatchWriter(statusUpdates)

	tickerDone := make(chan bool)
	ticker := time.NewTicker(10 * time.Second)

	// Ticker to get status updates from the server periodically
	var tickerWg sync.WaitGroup
	tickerWg.Add(1)
	go func() {
		// on return stop the ticker and close the status update channel
		defer func() {
			ticker.Stop()
			tickerWg.Done()
			log.Println("Stopped syncing manifest: ", manifest.Id)
		}()

		for {
			select {
			case <-tickerDone:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Checking status of files on server and verify.
				// This should return a list of files that have recently been finalized and then set the status of
				// those files to "Verified" on the server.
				log.Println("Verifying status for manifest: ", manifest.Id)
				err := s.ManifestService().VerifyFinalizedStatus(ctx, manifest, statusUpdates)
				if err != nil {
					log.Error("failed to verify manifest file statuses", err)
				}

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
		var uploadWg sync.WaitGroup

		session := uploadSession{
			manifestId: request.GetManifestId(),
			cancelFnc:  cancelFnc,
		}
		s.cancelFncs.Store(request.GetManifestId(), session)

		// Step 1 & 2. synchronize all files and wait for sync to complete
		s.syncProcessor(ctx, manifest)

		// Step 3. upload all files
		s.uploadProcessor(ctx, manifest, &uploadWg, statusUpdates)

		// continue syncing files for 15 minutes after uploads complete
		// make sure all upload workers have completed first
		syncTimerDelay := 15 * time.Minute
		if _, isPresent := s.syncCancelFncs.Load(manifest.Id); !isPresent {
			syncTimer := time.NewTimer(syncTimerDelay)
			defer syncTimer.Stop()
			cancel := make(chan struct{})
			s.syncCancelFncs.Store(manifest.Id, cancel)
			select {
			case <-syncTimer.C:
			case <-cancel:
			}
			uploadWg.Wait()
			tickerDone <- true
			tickerWg.Wait()
			close(statusUpdates)
			s.syncCancelFncs.Delete(manifest.Id)
		}
	}()

	response := pb.SimpleStatusResponse{Status: "Upload initiated."}
	return &response, nil
}

// batch write status updates and flush the batch every 5 seconds,
// or whenever the status updates channel has produced 100 status updates
func (s *agentServer) startStatusUpdateBatchWriter(statusUpdates <-chan models.UploadStatusUpdateMessage) {
	batch := make([]models.UploadStatusUpdateMessage, 0, 100)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		statusGroups := make(map[manifestFile.Status][]string)
		for _, msg := range batch {
			statusGroups[msg.Status] = append(statusGroups[msg.Status], msg.UploadID)
		}
		for status, uploadIds := range statusGroups {
			if err := s.ManifestService().BatchSetFileStatus(uploadIds, status); err != nil {
				log.Errorf("BatchSetFileStatus error for status %s (%d items): %v", status, len(uploadIds), err)
			}
		}
		batch = batch[:0]
	}

	statusUpdateTicker := time.NewTicker(5 * time.Second)
	defer statusUpdateTicker.Stop()

	for {
		select {
		case update, ok := <-statusUpdates:
			if !ok {
				flush()
				return
			}
			batch = append(batch, update)
			if len(batch) >= 100 {
				flush()
			}
		case <-statusUpdateTicker.C:
			flush()
		}
	}
}

func (s *agentServer) uploadProcessor(
	ctx context.Context,
	manifest *store.Manifest,
	uploadWg *sync.WaitGroup,
	statusUpdates chan<- models.UploadStatusUpdateMessage,
) {
	uploadWorkers := viper.GetInt("agent.upload_workers")
	chunkSize := viper.GetInt64("agent.upload_chunk_size")

	walker := make(chan store.ManifestFile, uploadWorkers)
	pennsieveClient, err := s.PennsieveClient()
	if err != nil {
		log.Error("Cannot get Pennsieve client")
	}

	// Database crawler to fetch rows
	go func() {
		defer close(walker)

		requestStatus := []manifestFile.Status{manifestFile.Registered}

		s.ManifestService().ManifestFilesToChannel(ctx, manifest.Id, requestStatus, walker)

	}()

	// Upload Handler
	go func() {
		cfg, err := config.LoadDefaultConfig(
			ctx,
			config.WithRegion("us-east-1"),
			config.WithCredentialsProvider(
				pennsieve.AWSCredentialProviderWithExpiration{
					AuthService: pennsieveClient.Authentication,
				},
			),
			config.WithCredentialsCacheOptions(func(o *aws.CredentialsCacheOptions) {
				o.ExpiryWindow = 5 * time.Minute
			}),
		)

		if err != nil {
			log.Fatal(err)
		}

		// For each file found walking, upload it to Amazon S3
		ctx, cancelFnc := context.WithCancel(ctx)
		session := uploadSession{
			manifestId: manifest.Id,
			cancelFnc:  cancelFnc,
		}
		s.cancelFncs.Store(manifest.Id, session)
		defer cancelFnc()

		// Create an S3 Client with the config
		s3Client := s3.NewFromConfig(cfg)

		// Create an uploader with the client and custom options
		uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
			u.PartSize = chunkSize * 1024 * 1024 // ...MB per part
		})

		s.updateSubscribers(0, 0, "", 0, pb.SubscribeResponse_UploadResponse_INIT)

		// Initiate the upload workers
		for worker := 1; worker <= uploadWorkers; worker++ {
			uploadWg.Add(1)
			workerId := int32(worker)
			go func(workerId int32) {
				log.Println("Starting upload worker: ", workerId)
				defer func() {
					log.Println("Closing upload worker: ", workerId)
					uploadWg.Done()
				}()

				err := s.uploadWorker(ctx, workerId, walker, statusUpdates, manifest.NodeId.String, uploader, cfg, pennsieveClient.GetAPIParams().UploadBucket, manifest.DatasetId, manifest.OrganizationId)
				if err != nil {
					log.Println("error in upload worker:", workerId, err)
				}
			}(workerId)
		}

		uploadWg.Wait()
		log.Println("Upload Completed")
		s.updateSubscribers(0, 0, "", 0, pb.SubscribeResponse_UploadResponse_COMPLETE)
		log.Println("Returned from uploader for manifest: ", manifest.Id)
	}()
}

func (s *agentServer) uploadWorker(
	ctx context.Context,
	workerId int32,
	jobs <-chan store.ManifestFile,
	statusUpdates chan<- models.UploadStatusUpdateMessage,
	manifestNodeId string,
	uploader *manager.Uploader,
	cfg aws.Config,
	uploadBucket string,
	datasetId string,
	organizationId string,
) error {

	for record := range jobs {
		file, err := os.Open(record.SourcePath)
		if err != nil {
			log.Println("Failed opening file", record.SourcePath, err)
			continue
		}

		fileInfo, err := file.Stat()
		if err != nil {
			log.Println("Failed describing file", record.SourcePath, err)
			continue
		}

		reader := &CustomReader{
			workerId: workerId,
			fp:       file,
			size:     fileInfo.Size(),
			signMap:  map[int64]struct{}{},
			s:        s,
		}

		s3Key := aws.String(fmt.Sprintf("%s/%s", manifestNodeId, record.UploadId))
		tags := fmt.Sprintf("OrgId=%s&DatasetId=%s", organizationId, datasetId)

		_, err = uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket:            aws.String(uploadBucket),
			Key:               s3Key,
			Body:              reader,
			ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
			Tagging:           &tags,
		})
		if err != nil {
			s.messageSubscribers("Upload Failed: see log for details.")

			// If Cancelled, need to manually abort upload on S3 to remove partial upload on S3. For other errors, this
			// is done automatically by the manager.
			if errors.Is(err, context.Canceled) {

				s.messageSubscribers("Upload canceled.")

				var mu manager.MultiUploadFailure
				if errors.As(err, &mu) {
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

				s.messageSubscribers("Upload Failed: see log for details.")

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

		statusUpdates <- models.UploadStatusUpdateMessage{
			UploadID: record.UploadId.String(),
			Status:   manifestFile.Uploaded,
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
	s        *agentServer
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
func (s *agentServer) updateSubscribers(
	total int64,
	current int64,
	name string,
	workerId int32,
	status pb.SubscribeResponse_UploadResponse_UploadStatus,
) {
	// A list of clients to unsubscribe in case of error
	var unsubscribe []int32

	// Iterate over all subscribers and send data to each client
	s.subscribers.Range(func(k, v any) bool {
		id, ok := k.(int32)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber key: %T", k))
			return false
		}
		sub, ok := v.(shared.Sub)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber value: %T", v))
			return false
		}
		// Send data over the gRPC Stream to the client
		if err := sub.Stream.Send(&pb.SubscribeResponse{
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
			case sub.Finished <- true:
				log.Info(fmt.Sprintf("Unsubscribed client: %d", id))
			default:
				log.Warn(fmt.Sprintf("Failed to send data to client: %v", err))
				// Default case is to avoid blocking in case client has already unsubscribed
			}
			// In case of error the client would re-subscribe so close the subscriber Stream
			unsubscribe = append(unsubscribe, id)
		}
		return true
	})

	// Unsubscribe erroneous client streams
	for _, id := range unsubscribe {
		s.subscribers.Delete(id)
	}
}

// sendCancelSubscribers Send Cancel Message to subscribers
func (s *agentServer) sendCancelSubscribers(message string) {
	// A list of clients to unsubscribe in case of error
	var unsubscribe []int32

	// Iterate over all subscribers and send data to each client
	s.subscribers.Range(func(k, v any) bool {
		id, ok := k.(int32)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber key: %T", k))
			return false
		}
		sub, ok := v.(shared.Sub)
		if !ok {
			log.Error(fmt.Sprintf("Failed to cast subscriber value: %T", v))
			return false
		}
		// Send data over the gRPC Stream to the client
		if err := sub.Stream.Send(&pb.SubscribeResponse{
			Type: pb.SubscribeResponse_UPLOAD_CANCEL,
			MessageData: &pb.SubscribeResponse_EventInfo{
				EventInfo: &pb.SubscribeResponse_EventResponse{Details: message}},
		}); err != nil {
			select {
			case sub.Finished <- true:
				log.Info(fmt.Sprintf("Unsubscribed client: %d", id))
			default:
				log.Warn(fmt.Sprintf("Failed to send data to client: %v", err))
				// Default case is to avoid blocking in case client has already unsubscribed
			}
			// In case of error the client would re-subscribe so close the subscriber Stream
			unsubscribe = append(unsubscribe, id)
		}
		return true
	})

	// Unsubscribe erroneous client streams
	for _, id := range unsubscribe {
		s.subscribers.Delete(id)
	}
}
