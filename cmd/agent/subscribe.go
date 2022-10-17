package agent

import (
	"github.com/pennsieve/pennsieve-agent/pkg/subscriber"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
		client, err := subscriber.NewSubscriberClient(int32(r1.Intn(100)))
		if err != nil {
			log.Fatal(err)
		}
		// Dispatch client goroutine
		go client.Start(nil, subscriber.StopOnStatus{
			Enable: false,
			OnType: nil,
		})

		time.Sleep(time.Second * 2)

		// The wait group purpose is to avoid exiting, the clients do not exit
		wg.Wait()
	},
}

func init() {
}
