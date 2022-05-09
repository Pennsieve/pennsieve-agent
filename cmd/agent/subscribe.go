package agent

import (
	"context"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/protos"
	"github.com/spf13/cobra"
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
	Short: "Subscribe to Agent messages",
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

func (c *longlivedClient) start() {
	var err error
	trackers := make(map[string]*mpb.Bar)
	pw := mpb.New()

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
			if t, ok := trackers[r.FileId]; ok {
				if r.GetCurrent() == r.GetTotal() {
					t.SetCurrent(r.GetCurrent())
					t.Abort(true)
					delete(trackers, r.FileId)
				} else {
					t.SetCurrent(r.GetCurrent())
				}

				//do something here
			} else {
				// New File
				if r.GetCurrent() == r.GetTotal() {
					t := pw.AddBar(r.GetTotal(),
						mpb.PrependDecorators(
							decor.Name(r.GetFileId()),
							decor.Percentage(decor.WCSyncSpace),
							//decor.OnComplete(decor.Name("\x1b[31minstalling\x1b[0m", decor.WCSyncSpaceR), "done!"),
							//decor.OnComplete(decor.EwmaETA(decor.ET_STYLE_MMSS, 0, decor.WCSyncWidth), ""),
						),
					)
					t.SetCurrent(r.GetCurrent())
					t.Abort(true)

				} else {
					t := pw.AddBar(r.GetTotal(),
						mpb.PrependDecorators(
							decor.Name(r.GetFileId()),
							decor.Percentage(decor.WCSyncSpace),
							//decor.OnComplete(decor.Name("\x1b[31minstalling\x1b[0m", decor.WCSyncSpaceR), "done!"),
							//decor.OnComplete(decor.EwmaETA(decor.ET_STYLE_MMSS, 0, decor.WCSyncWidth), ""),
						),
					)
					t.SetCurrent(r.GetCurrent())
					trackers[r.FileId] = t
				}

			}

		} else if response.GetType() == protos.SubsrcribeResponse_EVENT {
			info := response.GetEventInfo()
			fmt.Println(info.Details)
		} else if response.GetType() == protos.SubsrcribeResponse_UPLOAD_CANCEL {
			// Cancel all trackers.
			for _, p := range trackers {
				p.Abort(true)
			}

		} else {
			log.Println("Received an unknown message type: ", response.GetType())
		}

	}
}

// sleep is used to give the server time to unsubscribe the client and reset the stream
func (c *longlivedClient) sleep() {
	time.Sleep(time.Second * 5)
}

func mkConnection() (*grpc.ClientConn, error) {
	return grpc.Dial(":9058", []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock()}...)
}
