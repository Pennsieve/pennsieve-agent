package subscriber

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/viper"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
	"google.golang.org/grpc"
	"log"
	"time"
)

// longlivedClient holds the long lived gRPC client fields
type longlivedClient struct {
	client protos.AgentClient // client is the long lived gRPC client
	conn   *grpc.ClientConn   // conn is the client gRPC connection
	id     int32              // id is the client ID used for subscribing
}

// GetClient creates a new client instance
func GetClient(id int32) (*longlivedClient, error) {
	conn, err := mkConnection()
	if err != nil {
		return nil, err
	}
	return &longlivedClient{
		client: protos.NewAgentClient(conn),
		conn:   conn,
		id:     id,
	}, nil
}

// close is not used but is here as an example of how to close the gRPC client connection
func (c *longlivedClient) close() {
	if err := c.conn.Close(); err != nil {
		log.Fatal(err)
	}
}

// subscribe subscribes to messages from the gRPC server
func (c *longlivedClient) subscribe() (protos.Agent_SubscribeClient, error) {
	fmt.Printf("Subscribing to updates from Pennsieve Agent (id: %d)\n", c.id)
	return c.client.Subscribe(context.Background(), &protos.SubscribeRequest{Id: c.id})
}

// unsubscribe unsubscribes to messages from the gRPC server
func (c *longlivedClient) unsubscribe() error {
	fmt.Printf("Unsubscribing client ID %d", c.id)
	_, err := c.client.Unsubscribe(context.Background(), &protos.SubscribeRequest{Id: c.id})
	return err
}

type barInfo struct {
	bar    *mpb.Bar
	fileId string
}

func (c *longlivedClient) Start(types []protos.SubsrcribeResponse_MessageType, stopOnComplete bool) {
	var err error

	nrSyncedFiles := 0

	if types == nil {
		types = []protos.SubsrcribeResponse_MessageType{
			protos.SubsrcribeResponse_UPLOAD_STATUS,
			protos.SubsrcribeResponse_EVENT,
			protos.SubsrcribeResponse_SYNC_STATUS,
			protos.SubsrcribeResponse_UPLOAD_CANCEL,
		}
	}

	trackers := make(map[int32]barInfo)
	pw := mpb.New(
		mpb.PopCompletedMode())

	var syncBar *mpb.Bar

	// stream is the client side of the RPC stream
	var stream protos.Agent_SubscribeClient
	for {
		if stream == nil {
			if stream, err = c.subscribe(); err != nil {
				log.Printf("Failed to subscribe: %v", err)
				c.sleep()
				// Retry on failure
				continue
			}
		}
		response, err := stream.Recv()
		if err != nil {
			log.Printf("Failed to receive message: %v", err)
			// Clearing the stream will force the client to resubscribe on next iteration
			stream = nil
			c.sleep()
			// Retry on failure
			continue
		}

		switch response.GetType() {
		case protos.SubsrcribeResponse_UPLOAD_STATUS:
			if contains(types, protos.SubsrcribeResponse_UPLOAD_STATUS) {
				r := response.GetUploadStatus()

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
			}
		case protos.SubsrcribeResponse_EVENT:
			if contains(types, protos.SubsrcribeResponse_EVENT) {
				info := response.GetEventInfo()
				fmt.Println(info.Details)
			}
		case protos.SubsrcribeResponse_SYNC_STATUS:
			if contains(types, protos.SubsrcribeResponse_SYNC_STATUS) {
				info := response.GetSyncStatus()

				switch info.Status {
				case protos.SubsrcribeResponse_SyncResponse_INIT:
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

				case protos.SubsrcribeResponse_SyncResponse_IN_PROGRESS:
					nrSyncedFiles += int(info.NrSynced)
					syncBar.SetCurrent(int64(nrSyncedFiles))

				case protos.SubsrcribeResponse_SyncResponse_COMPLETE:
					if info.Status == protos.SubsrcribeResponse_SyncResponse_COMPLETE {
						syncBar.Completed()
						return
					}

				}

			}
		case protos.SubsrcribeResponse_UPLOAD_CANCEL:
			if contains(types, protos.SubsrcribeResponse_UPLOAD_CANCEL) {
				for _, p := range trackers {
					p.bar.Abort(true)
				}
			}
		default:
			log.Println("Received an unknown message type: ", response.GetType())
		}
	}
}

// https://play.golang.org/p/Qg_uv_inCek
// contains checks if a string is present in a slice
func contains(s []protos.SubsrcribeResponse_MessageType, str protos.SubsrcribeResponse_MessageType) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func (c *longlivedClient) addBar(pw *mpb.Progress, r *protos.SubsrcribeResponse_UploadResponse, trackers map[int32]barInfo) {
	t2 := pw.AddBar(r.GetTotal(),
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(
			decor.Name(r.GetFileId()),
			decor.Percentage(decor.WCSyncSpace),
		),
	)
	t2.SetCurrent(r.GetCurrent())

	info := barInfo{
		bar:    t2,
		fileId: r.GetFileId(),
	}

	trackers[r.WorkerId] = info
}

// sleep is used to give the server time to unsubscribe the client and reset the stream
func (c *longlivedClient) sleep() {
	time.Sleep(time.Second * 5)
}

func mkConnection() (*grpc.ClientConn, error) {
	port := viper.GetString("agent.port")
	return grpc.Dial(":"+port, []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock()}...)
}
