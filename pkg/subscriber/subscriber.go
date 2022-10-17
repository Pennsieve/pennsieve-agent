package subscriber

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/api/v1"
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
	client v1.AgentClient   // client is the long lived gRPC client
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
		client: v1.NewAgentClient(conn),
		conn:   conn,
		id:     id,
	}, nil
}

// StopOnStatus defines if the subscriber should unsubscribe and return on specific MessageType
type StopOnStatus struct {
	Enable bool
	OnType []v1.SubscribeResponse_MessageType
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
func (c *subscriberClient) subscribe() (v1.Agent_SubscribeClient, error) {
	fmt.Printf("Subscribing to updates from Pennsieve Agent (id: %d)\n", c.id)
	return c.client.Subscribe(context.Background(), &v1.SubscribeRequest{Id: c.id})
}

// unsubscribe deactivates listener for messages from the gRPC server
func (c *subscriberClient) unsubscribe() error {
	fmt.Printf("Unsubscribing client ID %d", c.id)
	_, err := c.client.Unsubscribe(context.Background(), &v1.SubscribeRequest{Id: c.id})
	return err
}

// Start listens to messages from the server and handles how to surface this to the user.
func (c *subscriberClient) Start(types []v1.SubscribeResponse_MessageType, stopOnComplete StopOnStatus) {
	var err error

	nrSyncedFiles := 0

	if types == nil {
		types = []v1.SubscribeResponse_MessageType{
			v1.SubscribeResponse_UPLOAD_STATUS,
			v1.SubscribeResponse_EVENT,
			v1.SubscribeResponse_SYNC_STATUS,
			v1.SubscribeResponse_UPLOAD_CANCEL,
		}
	}

	trackers := make(map[int32]barInfo)
	pw := mpb.New(
		mpb.PopCompletedMode())

	var syncBar *mpb.Bar

	// stream is the client side of the RPC stream
	var stream v1.Agent_SubscribeClient
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
		case v1.SubscribeResponse_UPLOAD_STATUS:
			if contains(types, v1.SubscribeResponse_UPLOAD_STATUS) {
				r := response.GetUploadStatus()

				switch r.Status {
				case v1.SubscribeResponse_UploadResponse_INIT:
				case v1.SubscribeResponse_UploadResponse_IN_PROGRESS:
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
				case v1.SubscribeResponse_UploadResponse_COMPLETE:
					if stopOnComplete.Enable && contains(stopOnComplete.OnType, v1.SubscribeResponse_UPLOAD_STATUS) {
						_ = c.unsubscribe()
						return
					}
				}
			}
		case v1.SubscribeResponse_EVENT:
			if contains(types, v1.SubscribeResponse_EVENT) {
				info := response.GetEventInfo()
				fmt.Println(info.Details)
			}
		case v1.SubscribeResponse_SYNC_STATUS:
			if contains(types, v1.SubscribeResponse_SYNC_STATUS) {
				info := response.GetSyncStatus()

				switch info.Status {
				case v1.SubscribeResponse_SyncResponse_INIT:
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

				case v1.SubscribeResponse_SyncResponse_IN_PROGRESS:
					nrSyncedFiles += int(info.NrSynced)
					syncBar.SetCurrent(int64(nrSyncedFiles))

				case v1.SubscribeResponse_SyncResponse_COMPLETE:
					if info.Status == v1.SubscribeResponse_SyncResponse_COMPLETE {
						syncBar.Completed()
						if stopOnComplete.Enable && contains(stopOnComplete.OnType, v1.SubscribeResponse_SYNC_STATUS) {
							_ = c.unsubscribe()
							return
						}

					}

				}

			}
		case v1.SubscribeResponse_UPLOAD_CANCEL:
			if contains(types, v1.SubscribeResponse_UPLOAD_CANCEL) {
				for _, p := range trackers {
					p.bar.Abort(true)
				}
			}
		default:
			log.Error("Received an unknown message type: ", response.GetType())
		}
	}
}

// contains checks if a string is present in a slice
func contains(s []v1.SubscribeResponse_MessageType, str v1.SubscribeResponse_MessageType) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// addBar adds a progress bar to the trackers map
func (c *subscriberClient) addBar(pw *mpb.Progress, r *v1.SubscribeResponse_UploadResponse, trackers map[int32]barInfo) {
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
