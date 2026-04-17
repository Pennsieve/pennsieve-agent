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

		// Run sync and upload in parallel. Sync flushes each 250-file batch
		// to SQLite as the server confirms it, so upload workers can start
		// picking up Registered files seconds after UploadManifest is
		// invoked — rather than waiting for the entire manifest (potentially
		// 100k+ files) to register up-front. Closes syncDone when sync
		// completes so the upload walker knows it's safe to stop polling
		// for newly-registered rows.
		syncDone := make(chan struct{})
		go func() {
			defer close(syncDone)
			s.syncProcessor(ctx, manifest)
		}()

		// Step 3. upload all files
		s.uploadProcessor(ctx, manifest, &uploadWg, statusUpdates, syncDone)

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
	syncDone <-chan struct{},
) {
	uploadWorkers := viper.GetInt("agent.upload_workers")
	chunkSize := viper.GetInt64("agent.upload_chunk_size")

	walker := make(chan store.ManifestFile, uploadWorkers)
	pennsieveClient, err := s.PennsieveClient()
	if err != nil {
		log.Error("Cannot get Pennsieve client")
	}

	// Polling walker. Re-queries SQLite for Registered rows every pollInterval
	// so uploads can start as soon as the first sync batch lands, without
	// waiting for the whole manifest to register. Emits each uploadId at most
	// once via an in-memory claim set — sync flushes to SQLite faster than
	// upload-worker status writes, so the same row could otherwise be
	// re-queried before its status transitions out of Registered.
	go func() {
		defer close(walker)

		const pollInterval = 1 * time.Second
		claimed := make(map[string]struct{})
		requestStatus := []manifestFile.Status{manifestFile.Registered}

		drain := func() int {
			tmp := make(chan store.ManifestFile, 256)
			go func() {
				defer close(tmp)
				s.ManifestService().ManifestFilesToChannel(ctx, manifest.Id, requestStatus, tmp)
			}()
			n := 0
			for mf := range tmp {
				uid := mf.UploadId.String()
				if _, ok := claimed[uid]; ok {
					continue
				}
				claimed[uid] = struct{}{}
				select {
				case walker <- mf:
					n++
				case <-ctx.Done():
					return n
				}
			}
			return n
		}

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		syncFinished := false
		for {
			emitted := drain()
			if syncFinished && emitted == 0 {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-syncDone:
				syncFinished = true
				// Loop once more to drain anything written by the final sync
				// batch, then exit if no new rows appear.
			case <-ticker.C:
			}
		}
	}()

	// Try to get direct-to-storage credentials. If the endpoint returns 404
	// (older server), silently fall back to the Cognito + upload-bucket path
	// so old agents keep working against old servers and vice versa.
	directCreds, directMode := s.tryGetDirectStorageCredentials(ctx, pennsieveClient, manifest)

	// Upload Handler
	go func() {
		var cfg aws.Config
		var bucket, keyPrefix string

		if directMode {
			c, err := config.LoadDefaultConfig(
				ctx,
				config.WithRegion(directCreds.Region()),
				config.WithCredentialsProvider(directCreds),
				config.WithCredentialsCacheOptions(func(o *aws.CredentialsCacheOptions) {
					o.ExpiryWindow = 5 * time.Minute
				}),
			)
			if err != nil {
				log.Fatal(err)
			}
			cfg = c
			bucket, keyPrefix = directCreds.BucketAndPrefix()
			log.Infof("Direct-to-storage upload enabled: bucket=%s prefix=%s", bucket, keyPrefix)
		} else {
			c, err := config.LoadDefaultConfig(
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
			cfg = c
			bucket = pennsieveClient.GetAPIParams().UploadBucket
			// Legacy path keys by manifestNodeId — no O{org}/D{ds} prefix.
			keyPrefix = manifest.NodeId.String
			log.Info("Direct-to-storage unavailable, using legacy upload-bucket path")
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

		// Direct-to-storage path: pipe successful uploads through a finalize
		// batcher that reports completion to the server (two-phase upload).
		// Legacy path has no finalize channel — the upload lambda handles it
		// via S3 events.
		var finalizeCh chan finalizeJob
		var finalizeWg sync.WaitGroup
		if directMode {
			finalizeCh = make(chan finalizeJob, 1000)
			finalizeWg.Add(1)
			go func() {
				defer finalizeWg.Done()
				s.finalizeBatcher(ctx, pennsieveClient, manifest, finalizeCh, statusUpdates)
			}()
		}

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

				err := s.uploadWorker(ctx, workerId, walker, statusUpdates, finalizeCh, keyPrefix, uploader, cfg, bucket, manifest.DatasetId, manifest.OrganizationId)
				if err != nil {
					log.Println("error in upload worker:", workerId, err)
				}
			}(workerId)
		}

		uploadWg.Wait()
		if finalizeCh != nil {
			close(finalizeCh)
			finalizeWg.Wait()
		}
		log.Println("Upload Completed")
		s.updateSubscribers(0, 0, "", 0, pb.SubscribeResponse_UploadResponse_COMPLETE)
		log.Println("Returned from uploader for manifest: ", manifest.Id)
	}()
}

// finalizeJob is one upload-complete event fed to the finalize batcher.
type finalizeJob struct {
	UploadID string
	Size     int64
	SHA256   string
}

// tryGetDirectStorageCredentials attempts to obtain STS credentials scoped to
// the manifest's destination storage bucket. Returns (nil, false) on HTTP 404
// so the caller can fall back to the Cognito path. Other errors are logged
// and treated as fallback as well — the legacy path remains fully functional.
func (s *agentServer) tryGetDirectStorageCredentials(
	ctx context.Context,
	client *pennsieve.Client,
	manifest *store.Manifest,
) (*pennsieve.StorageCredentialsProvider, bool) {
	if !manifest.NodeId.Valid {
		return nil, false
	}
	provider := &pennsieve.StorageCredentialsProvider{
		Manifest:       client.Manifest,
		DatasetID:      manifest.DatasetId,
		ManifestNodeID: manifest.NodeId.String,
	}
	// Force an initial fetch so we can detect 404 before spinning up workers.
	if _, err := provider.Retrieve(ctx); err != nil {
		var httpErr *pennsieve.HTTPError
		if errors.As(err, &httpErr) && (httpErr.StatusCode == 404 || httpErr.StatusCode == 409) {
			return nil, false
		}
		log.Warnf("storage-credentials fetch failed, falling back: %v", err)
		return nil, false
	}
	return provider, true
}

// finalizeBatcher groups successful uploads into bounded batches (500 files or
// 5 seconds) and calls POST /manifest/files/finalize. Per-file success becomes
// a local Finalized status update; per-file failures keep local status at
// Uploaded so they are retried on the next agent run.
func (s *agentServer) finalizeBatcher(
	ctx context.Context,
	client *pennsieve.Client,
	manifest *store.Manifest,
	in <-chan finalizeJob,
	statusUpdates chan<- models.UploadStatusUpdateMessage,
) {
	const maxBatch = 500
	const flushInterval = 5 * time.Second

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	batch := make([]pennsieve.FinalizeFile, 0, maxBatch)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		s.callFinalize(ctx, client, manifest, batch, statusUpdates)
		batch = batch[:0]
	}

	for {
		select {
		case job, ok := <-in:
			if !ok {
				flush()
				return
			}
			batch = append(batch, pennsieve.FinalizeFile{
				UploadID: job.UploadID,
				Size:     job.Size,
				SHA256:   job.SHA256,
			})
			if len(batch) >= maxBatch {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		}
	}
}

// callFinalize executes one POST /manifest/files/finalize call and persists
// the per-file response directly to the local SQLite store. Writes directly
// (rather than through statusUpdates channel) so there's no chance of the
// later Finalized status landing in the same batch as the earlier Uploaded
// status for the same uploadId.
func (s *agentServer) callFinalize(
	ctx context.Context,
	client *pennsieve.Client,
	manifest *store.Manifest,
	batch []pennsieve.FinalizeFile,
	statusUpdates chan<- models.UploadStatusUpdateMessage,
) {
	resp, err := client.Manifest.FinalizeManifestFiles(ctx, manifest.DatasetId, manifest.NodeId.String, batch)
	if err != nil {
		// Whole-batch failure — leave local status at Uploaded so files are
		// retried on the next agent restart via ResetStatusForManifest /
		// VerifyFinalizedStatus plumbing. Surface via log.
		log.Errorf("finalize batch failed (%d files): %v", len(batch), err)
		return
	}

	var finalizedIDs []string
	for _, r := range resp.Results {
		switch r.Status {
		case "finalized":
			finalizedIDs = append(finalizedIDs, r.UploadID)
		default: // "failed"
			log.Warnf("finalize failed for upload %s: %s", r.UploadID, r.Error)
			// Leave local status as Uploaded → retried on next run.
		}
	}
	if len(finalizedIDs) > 0 {
		if err := s.ManifestService().BatchSetFileStatus(finalizedIDs, manifestFile.Finalized); err != nil {
			log.Errorf("BatchSetFileStatus Finalized (%d items): %v", len(finalizedIDs), err)
		}
	}
}

func (s *agentServer) uploadWorker(
	ctx context.Context,
	workerId int32,
	jobs <-chan store.ManifestFile,
	statusUpdates chan<- models.UploadStatusUpdateMessage,
	finalizeCh chan<- finalizeJob,
	keyPrefix string,
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

		// keyPrefix is either the manifest node id (legacy upload bucket) or
		// O{orgId}/D{datasetId}/{manifestId} (direct-to-storage). Both use
		// {prefix}/{uploadId} as the object key.
		s3Key := aws.String(fmt.Sprintf("%s/%s", keyPrefix, record.UploadId))
		tags := fmt.Sprintf("OrgId=%s&DatasetId=%s", organizationId, datasetId)

		uploadOut, err := uploader.Upload(ctx, &s3.PutObjectInput{
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

		// Direct-to-storage: hand off to the finalize batcher so the server
		// can create Postgres rows and mark the file Finalized. Legacy path
		// has no finalize channel — the upload lambda handles it via S3 events.
		if finalizeCh != nil {
			// SHA256 is required on finalize so the server can verify
			// content integrity against what S3 stored. For multipart
			// uploads this is the checksum-of-checksums (suffixed with
			// the part count, e.g. "base64==-42"); the server compares
			// it to HEAD's ChecksumSHA256, so this round-trip is exact.
			var sha256 string
			if uploadOut != nil && uploadOut.ChecksumSHA256 != nil {
				sha256 = *uploadOut.ChecksumSHA256
			}
			finalizeCh <- finalizeJob{
				UploadID: record.UploadId.String(),
				Size:     fileInfo.Size(),
				SHA256:   sha256,
			}
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
