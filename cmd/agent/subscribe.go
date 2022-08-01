package agent

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
	"google.golang.org/grpc"
	"log"
	"math/rand"
	"sync"
	"time"
)

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to messages from the Pennsieve Agent.",
	Long: `Open long-lived connection to the server server and visualize messages from the server.

`,
	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup

		wg.Add(1)
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		client, err := mkAgentClient(int32(r1.Intn(100)))
		if err != nil {
			log.Fatal(err)
		}
		// Dispatch client goroutine
		go client.start()
		time.Sleep(time.Second * 2)

		// The wait group purpose is to avoid exiting, the clients do not exit
		wg.Wait()

	},
}

func init() {
}

// longlivedClient holds the long lived gRPC client fields
type longlivedClient struct {
	client protos.AgentClient // client is the long lived gRPC client
	conn   *grpc.ClientConn   // conn is the client gRPC connection
	id     int32              // id is the client ID used for subscribing
}

// mkLonglivedClient creates a new client instance
func mkAgentClient(id int32) (*longlivedClient, error) {
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
	log.Printf("Subscribing client ID: %d", c.id)
	return c.client.Subscribe(context.Background(), &protos.SubscribeRequest{Id: c.id})
}

// unsubscribe unsubscribes to messages from the gRPC server
func (c *longlivedClient) unsubscribe() error {
	log.Printf("Unsubscribing client ID %d", c.id)
	_, err := c.client.Unsubscribe(context.Background(), &protos.SubscribeRequest{Id: c.id})
	return err
}

type barInfo struct {
	bar    *mpb.Bar
	fileId string
}

func (c *longlivedClient) start() {
	var err error

	trackers := make(map[int32]barInfo)
	pw := mpb.New(
		mpb.PopCompletedMode())

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

		if response.GetType() == protos.SubsrcribeResponse_UPLOAD_STATUS {
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

		} else if response.GetType() == protos.SubsrcribeResponse_EVENT {
			info := response.GetEventInfo()
			fmt.Println(info.Details)
		} else if response.GetType() == protos.SubsrcribeResponse_UPLOAD_CANCEL {
			// Cancel all trackers.
			for _, p := range trackers {
				p.bar.Abort(true)
			}

		} else {
			log.Println("Received an unknown message type: ", response.GetType())
		}

	}
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
