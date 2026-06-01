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
	"github.com/pennsieve/pennsieve-agent/v2/pkg/models"
	"github.com/pennsieve/pennsieve-agent/v2/pkg/shared"
	"github.com/pennsieve/pennsieve-agent/v2/pkg/store"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/manifest/manifestFile"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	pb "github.com/pennsieve/pennsieve-agent/v2/api/v1"
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

	// Post-upload reconciliation between local state and server-side
	// Finalized status is handled by pkg/reconciler, which runs for the
	// lifetime of the agent daemon — not bolted into the upload-session
	// goroutine, where its context dies the moment this handler returns.

	// Manager:
	// 1. Create thread to sync
	// 2. Wait for sync to be done
	// 3. Create thread for upload
	//
	// uploadProcessor takes ownership of statusUpdates from here on — it
	// closes the channel inside the upload-handler goroutine after workers
	// finish and the finalize batcher drains. The manager goroutine is
	// fire-and-forget; if it tried to wait+close itself, the wait group it
	// observed would be empty (uploadWg.Add happens inside the
	// upload-handler goroutine that uploadProcessor spawns), so the close
	// would race ahead of worker sends and panic.
	go func() {
		// Create Context and store cancel function in server object.
		ctx, cancelFnc := context.WithCancel(context.Background())

		session := uploadSession{
			manifestId: request.GetManifestId(),
			cancelFnc:  cancelFnc,
		}
		s.cancelFncs.Store(request.GetManifestId(), session)

		// Obtain the manifest node id up front and wrap into a syncedManifest.
		// Having a non-empty node id is a prerequisite for the credential
		// fetch and the finalize calls; by making syncedManifest the only
		// type those functions accept, the compiler enforces the invariant
		// that callers have registered the manifest first.
		syncedM, err := s.prepareForUpload(manifest)
		if err != nil {
			log.Errorf("prepareForUpload: %v", err)
			close(statusUpdates) // batch writer goroutine would otherwise leak
			return
		}

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
		s.uploadProcessor(ctx, syncedM, statusUpdates, syncDone, request.GetOnConflict())
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

// uploadProcessor takes ownership of statusUpdates and is responsible for
// closing it when no more writes can happen (after workers finish and the
// finalize batcher drains). All early-error paths must close it too — the
// batch-writer goroutine on the other end blocks on it and would otherwise
// leak.
func (s *agentServer) uploadProcessor(
	ctx context.Context,
	manifest *syncedManifest,
	statusUpdates chan<- models.UploadStatusUpdateMessage,
	syncDone <-chan struct{},
	onConflict string,
) {
	// Re-read config so changes to config.ini take effect without restarting the daemon.
	// viper.Set values (e.g. from PENNSIEVE_AGENT_UPLOAD_WORKERS at daemon start) still
	// take precedence over config-file values — their priority is unaffected by ReadInConfig.
	_ = viper.ReadInConfig()
	uploadWorkers := viper.GetInt("agent.upload_workers")
	chunkSize := viper.GetInt64("agent.upload_chunk_size")

	pennsieveClient, err := s.PennsieveClient()
	if err != nil {
		log.Errorf("Cannot get Pennsieve client: %v", err)
		close(statusUpdates)
		return
	}

	// Fetch credentials before launching the walker so any failure aborts
	// cleanly rather than leaking the walker goroutine.
	creds, err := s.getStorageCredentials(ctx, pennsieveClient, manifest)
	if err != nil {
		log.Errorf("get storage credentials: %v", err)
		s.messageSubscribers("Upload failed: unable to obtain storage credentials.")
		close(statusUpdates)
		return
	}

	walker := make(chan store.ManifestFile, uploadWorkers)

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
				s.ManifestService().ManifestFilesToChannel(ctx, manifest.ID(), requestStatus, tmp)
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

	// Upload Handler
	go func() {
		cfg, err := config.LoadDefaultConfig(
			ctx,
			config.WithRegion(creds.Region()),
			config.WithCredentialsProvider(creds),
			config.WithCredentialsCacheOptions(func(o *aws.CredentialsCacheOptions) {
				o.ExpiryWindow = 5 * time.Minute
			}),
		)
		if err != nil {
			log.Fatal(err)
		}
		bucket, keyPrefix := creds.BucketAndPrefix()
		log.Infof("Direct-to-storage upload: bucket=%s prefix=%s", bucket, keyPrefix)

		// For each file found walking, upload it to Amazon S3
		ctx, cancelFnc := context.WithCancel(ctx)
		session := uploadSession{
			manifestId: manifest.ID(),
			cancelFnc:  cancelFnc,
		}
		s.cancelFncs.Store(manifest.ID(), session)
		defer cancelFnc()

		// Create an S3 Client with the config
		s3Client := s3.NewFromConfig(cfg)

		// Create an uploader with the client and custom options
		uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
			u.PartSize = chunkSize * 1024 * 1024 // ...MB per part
		})

		s.updateSubscribers(0, 0, "", 0, pb.SubscribeResponse_UploadResponse_INIT)

		// Pipe successful uploads through a finalize batcher that reports
		// completion to the server (two-phase upload).
		finalizeCh := make(chan finalizeJob, 1000)
		var finalizeWg sync.WaitGroup
		finalizeWg.Add(1)
		go func() {
			defer finalizeWg.Done()
			s.finalizeBatcher(ctx, pennsieveClient, manifest, finalizeCh, statusUpdates, onConflict)
		}()

		// Initiate the upload workers. uploadWg is local to this goroutine so
		// the wait group is fully populated before any Wait() can observe it
		// — earlier wiring put the Wait in the caller goroutine, which would
		// race ahead of the Add()s here and let close(statusUpdates) fire
		// while workers were still sending.
		var uploadWg sync.WaitGroup
		for worker := 1; worker <= uploadWorkers; worker++ {
			uploadWg.Add(1)
			workerId := int32(worker)
			go func(workerId int32) {
				log.Println("Starting upload worker: ", workerId)
				defer func() {
					log.Println("Closing upload worker: ", workerId)
					uploadWg.Done()
				}()

				err := s.uploadWorker(ctx, workerId, walker, statusUpdates, finalizeCh, keyPrefix, uploader, cfg, bucket, manifest.DatasetID(), manifest.OrganizationID())
				if err != nil {
					log.Println("error in upload worker:", workerId, err)
				}
			}(workerId)
		}

		uploadWg.Wait()
		close(finalizeCh)
		finalizeWg.Wait()
		// All writers to statusUpdates (workers + finalize batcher) have
		// stopped — safe to close so the batch writer drains and exits.
		close(statusUpdates)
		log.Println("Upload Completed")
		s.updateSubscribers(0, 0, "", 0, pb.SubscribeResponse_UploadResponse_COMPLETE)
		log.Println("Returned from uploader for manifest: ", manifest.ID())
	}()
}

// finalizeJob is one upload-complete event fed to the finalize batcher.
type finalizeJob struct {
	UploadID string
	Size     int64
	SHA256   string
}

// syncedManifest wraps a store.Manifest with the compile-time guarantee that
// the manifest has been registered with the server and has a non-empty node
// id. The upload flow's credential fetch, key-prefix construction, and
// finalize calls all depend on that invariant; encoding it in the type rules
// out a whole class of "forgot to call getCreateManifestId first" bugs.
//
// Construct only via prepareForUpload — the struct fields are private so code
// can't bypass that path.
type syncedManifest struct {
	id             int32
	nodeID         string // guaranteed non-empty
	datasetID      string
	organizationID string
	store          *store.Manifest
}

func (m *syncedManifest) ID() int32              { return m.id }
func (m *syncedManifest) NodeID() string         { return m.nodeID }
func (m *syncedManifest) DatasetID() string      { return m.datasetID }
func (m *syncedManifest) OrganizationID() string { return m.organizationID }

// Underlying returns the backing store.Manifest for code that still needs the
// raw struct (notably syncProcessor, which also runs getCreateManifestId as a
// no-op on the already-synced pointer).
func (m *syncedManifest) Underlying() *store.Manifest { return m.store }

// prepareForUpload registers the manifest with the server if needed and
// returns a syncedManifest, or an error if registration fails or returns an
// empty node id. This is the only supported construction path for
// syncedManifest; callers that operate on upload-related flows (credential
// fetch, key construction, finalize) must go through it.
func (s *agentServer) prepareForUpload(m *store.Manifest) (*syncedManifest, error) {
	if err := s.getCreateManifestId(m); err != nil {
		return nil, fmt.Errorf("get manifest node id: %w", err)
	}
	if !m.NodeId.Valid || m.NodeId.String == "" {
		return nil, errors.New("manifest node id still empty after getCreateManifestId")
	}
	return &syncedManifest{
		id:             m.Id,
		nodeID:         m.NodeId.String,
		datasetID:      m.DatasetId,
		organizationID: m.OrganizationId,
		store:          m,
	}, nil
}

// getStorageCredentials returns STS credentials scoped to the manifest's
// destination storage bucket, force-fetching once so callers can surface
// failures before spinning up upload workers.
func (s *agentServer) getStorageCredentials(
	ctx context.Context,
	client *pennsieve.Client,
	manifest *syncedManifest,
) (*pennsieve.StorageCredentialsProvider, error) {
	provider := &pennsieve.StorageCredentialsProvider{
		Manifest:       client.Manifest,
		DatasetID:      manifest.DatasetID(),
		ManifestNodeID: manifest.NodeID(),
	}
	if _, err := provider.Retrieve(ctx); err != nil {
		return nil, fmt.Errorf("retrieve storage credentials: %w", err)
	}
	return provider, nil
}

// finalizeBatcher groups successful uploads into bounded batches (500 files or
// 5 seconds) and calls POST /manifest/files/finalize. Per-file success becomes
// a local Finalized status update; per-file failures keep local status at
// Uploaded so they are retried on the next agent run.
func (s *agentServer) finalizeBatcher(
	ctx context.Context,
	client *pennsieve.Client,
	manifest *syncedManifest,
	in <-chan finalizeJob,
	statusUpdates chan<- models.UploadStatusUpdateMessage,
	onConflict string,
) {
	// Keep in sync with maxFinalizeBatch in the upload-service-v2 finalize
	// handler and the OpenAPI spec's maxItems — the server rejects oversized
	// batches with 400.
	const maxBatch = 250
	const flushInterval = 5 * time.Second

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	batch := make([]pennsieve.FinalizeFile, 0, maxBatch)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		s.callFinalize(ctx, client, manifest, batch, statusUpdates, onConflict)
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
	manifest *syncedManifest,
	batch []pennsieve.FinalizeFile,
	statusUpdates chan<- models.UploadStatusUpdateMessage,
	onConflict string,
) {
	var opts []pennsieve.FinalizeOption
	if onConflict != "" {
		opts = append(opts, pennsieve.WithOnConflict(onConflict))
	}
	resp, err := client.Manifest.FinalizeManifestFiles(ctx, manifest.DatasetID(), manifest.NodeID(), batch, opts...)
	if err != nil {
		// Whole-batch failure — leave local status at Uploaded so the
		// reconciler can converge it once the server reports Finalized
		// (or so the user can ResetStatusForManifest and retry).
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

		// keyPrefix is O{orgId}/D{datasetId}/{manifestId}; the object key is
		// {prefix}/{uploadId}.
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

		// SHA256 is required on finalize so the server can verify content
		// integrity against what S3 stored. For multipart uploads this is the
		// checksum-of-checksums (suffixed with the part count, e.g.
		// "base64==-42"); the server compares it to HEAD's ChecksumSHA256, so
		// this round-trip is exact.
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
