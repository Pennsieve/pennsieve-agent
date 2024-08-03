package subscriber

import (
	"context"
	"fmt"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

// subscriberClient holds the long-lived gRPC client fields
type subscriberClient struct {
	client api.AgentClient  // client is the long lived gRPC client
	conn   *grpc.ClientConn // conn is the client gRPC connection
	id     int32            // id is the client ID used for subscribing
}

// NewSubscriberClient creates a new client instance
func NewSubscriberClient(id int32) (*subscriberClient, error) {
	port := viper.GetString("agent.port")
	conn, err := grpc.Dial(":"+port, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock()}...)

	if err != nil {
		return nil, err
	}
	return &subscriberClient{
		client: api.NewAgentClient(conn),
		conn:   conn,
		id:     id,
	}, nil
}

// StopOnStatus defines if the subscriber should unsubscribe and return on specific MessageType
type StopOnStatus struct {
	Enable bool
	OnType []api.SubscribeResponse_MessageType
}

// barInfo maps a progress-bar to a specific file
type barInfo struct {
	bar    *mpb.Bar
	fileId string
}

// close is not used but is here as an example of how to close the gRPC client connection
func (c *subscriberClient) close() {
	if err := c.conn.Close(); err != nil {
		log.Fatal(err)
	}
}

// subscribe activates the client to listen for messages from the gRPC server
func (c *subscriberClient) subscribe() (api.Agent_SubscribeClient, error) {
	fmt.Printf("Subscribing to updates from Pennsieve Agent (id: %d)\n", c.id)
	return c.client.Subscribe(context.Background(), &api.SubscribeRequest{Id: c.id})
}

// unsubscribe deactivates listener for messages from the gRPC server
func (c *subscriberClient) unsubscribe() error {
	fmt.Printf("Unsubscribing client ID %d\n", c.id)
	_, err := c.client.Unsubscribe(context.Background(), &api.SubscribeRequest{Id: c.id})
	return err
}

// Start listens to messages from the server and handles how to surface this to the user.
func (c *subscriberClient) Start(types []api.SubscribeResponse_MessageType, stopOnComplete StopOnStatus) {
	var err error

	nrSyncedFiles := 0

	if types == nil {
		types = []api.SubscribeResponse_MessageType{
			api.SubscribeResponse_UPLOAD_STATUS,
			api.SubscribeResponse_EVENT,
			api.SubscribeResponse_SYNC_STATUS,
			api.SubscribeResponse_UPLOAD_CANCEL,
			api.SubscribeResponse_DOWNLOAD_STATUS,
		}
	}

	trackers := make(map[int32]barInfo)
	downloadTrackers := make(map[string]barInfo)
	pw := mpb.New(
		mpb.PopCompletedMode())

	var syncBar *mpb.Bar

	// stream is the client side of the RPC stream
	var stream api.Agent_SubscribeClient
	for {
		if stream == nil {
			if stream, err = c.subscribe(); err != nil {
				log.WithFields(log.Fields{
					"error": err.Error(),
				}).Warn("Failed to subscribe.")
				c.sleep()
				// Retry on failure
				continue
			}
		}
		response, err := stream.Recv()
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to receive message: %v", err))
			// Clearing the stream will force the client to resubscribe on next iteration
			stream = nil
			c.sleep()
			// Retry on failure
			continue
		}

		switch response.GetType() {
		case api.SubscribeResponse_UPLOAD_STATUS:
			if contains(types, api.SubscribeResponse_UPLOAD_STATUS) {
				r := response.GetUploadStatus()

				switch r.Status {
				case api.SubscribeResponse_UploadResponse_INIT:
				case api.SubscribeResponse_UploadResponse_IN_PROGRESS:
					// Get/Create Tracker
					if t, ok := trackers[r.WorkerId]; ok {
						if t.fileId == r.FileId {
							t.bar.SetCurrent(r.GetCurrent())
							t.bar.SetPriority(int(100 / (float32(r.Current) / float32(r.Total))))
						} else {
							// In this case the previous upload was completed and should have been popped.
							c.addBar(pw, r, trackers)
						}

					} else {
						// initialization of new worker progress bar.
						c.addBar(pw, r, trackers)
					}
				case api.SubscribeResponse_UploadResponse_COMPLETE:
					if stopOnComplete.Enable && contains(stopOnComplete.OnType, api.SubscribeResponse_UPLOAD_STATUS) {
						_ = c.unsubscribe()
						return
					}
				}
			}
		case api.SubscribeResponse_EVENT:
			if contains(types, api.SubscribeResponse_EVENT) {
				info := response.GetEventInfo()
				fmt.Println(info.Details)
			}
		case api.SubscribeResponse_SYNC_STATUS:
			if contains(types, api.SubscribeResponse_SYNC_STATUS) {
				info := response.GetSyncStatus()

				if syncBar == nil {
					nrSyncedFiles = 0
					syncBar = pw.AddBar(info.Total,
						mpb.BarFillerClearOnComplete(),
						mpb.PrependDecorators(
							decor.Name("Sync Progress"),
							decor.Percentage(decor.WCSyncSpace),
						),
					)
					if info.Total == 0 {
						syncBar.SetTotal(1, true)
						nrSyncedFiles += 1
					} else {
						syncBar.SetCurrent(int64(nrSyncedFiles))
					}
				}

				switch info.Status {
				case api.SubscribeResponse_SyncResponse_INIT:

				case api.SubscribeResponse_SyncResponse_IN_PROGRESS:
					nrSyncedFiles += int(info.NrSynced)
					syncBar.SetCurrent(int64(nrSyncedFiles))

				case api.SubscribeResponse_SyncResponse_COMPLETE:
					if info.Status == api.SubscribeResponse_SyncResponse_COMPLETE {
						syncBar.Completed()
						if stopOnComplete.Enable && contains(stopOnComplete.OnType, api.SubscribeResponse_SYNC_STATUS) {
							_ = c.unsubscribe()
							return
						}

					}

				}

			}
		case api.SubscribeResponse_UPLOAD_CANCEL:
			if contains(types, api.SubscribeResponse_UPLOAD_CANCEL) {
				for _, p := range trackers {
					p.bar.Abort(true)
				}
			}
		case api.SubscribeResponse_DOWNLOAD_CANCEL:
			if contains(types, api.SubscribeResponse_DOWNLOAD_CANCEL) {
				for _, p := range downloadTrackers {
					p.bar.Abort(true)
				}
			}
		case api.SubscribeResponse_DOWNLOAD_STATUS:
			if contains(types, api.SubscribeResponse_DOWNLOAD_STATUS) {
				r := response.GetDownloadStatus()

				switch r.Status {
				case api.SubscribeResponse_DownloadStatusResponse_INIT:
				case api.SubscribeResponse_DownloadStatusResponse_IN_PROGRESS:
					// Get/Create Tracker
					if t, ok := downloadTrackers[r.FileId]; ok {
						if t.fileId == r.FileId {
							t.bar.SetCurrent(r.GetCurrent())
							t.bar.SetPriority(int(100 / (float32(r.Current) / float32(r.Total))))
						} else {
							// In this case the previous upload was completed and should have been popped.
							c.addDownloadBar(pw, r, downloadTrackers)
						}

					} else {
						// initialization of new worker progress bar.
						c.addDownloadBar(pw, r, downloadTrackers)
					}
				case api.SubscribeResponse_DownloadStatusResponse_COMPLETE:
					if stopOnComplete.Enable && contains(stopOnComplete.OnType, api.SubscribeResponse_DOWNLOAD_STATUS) {
						_ = c.unsubscribe()
						return
					}
				}

			}

		default:
			log.Error("Received an unknown message type: ", response.GetType())
		}
	}
}

// contains checks if a string is present in a slice
func contains(s []api.SubscribeResponse_MessageType, str api.SubscribeResponse_MessageType) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// addBar adds a progress bar to the trackers map
func (c *subscriberClient) addDownloadBar(pw *mpb.Progress, r *api.SubscribeResponse_DownloadStatusResponse, trackers map[string]barInfo) {
	newBar := pw.AddBar(r.GetTotal(),
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(
			decor.Name(r.GetFileId()),
			decor.NewPercentage("% .1f", decor.WCSyncSpace),
		),
	)
	newBar.SetCurrent(r.GetCurrent())

	info := barInfo{
		bar:    newBar,
		fileId: r.GetFileId(),
	}

	trackers[r.FileId] = info
}

// addBar adds a progress bar to the trackers map
func (c *subscriberClient) addBar(pw *mpb.Progress, r *api.SubscribeResponse_UploadResponse, trackers map[int32]barInfo) {
	newBar := pw.AddBar(r.GetTotal(),
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(
			decor.Name(r.GetFileId()),
			decor.Percentage(decor.WCSyncSpace),
		),
	)
	newBar.SetCurrent(r.GetCurrent())

	info := barInfo{
		bar:    newBar,
		fileId: r.GetFileId(),
	}

	trackers[r.WorkerId] = info
}

// sleep is used to give the server time to unsubscribe the client and reset the stream
func (c *subscriberClient) sleep() {
	time.Sleep(time.Second * 5)
}
