package shared

import pb "github.com/pennsieve/pennsieve-agent/api/v1"

type Sub struct {
	Stream   pb.Agent_SubscribeServer // Stream is the server side of the RPC Stream
	Finished chan<- bool              //finishedd is used to signal closure of a client subscribing goroutine
}
