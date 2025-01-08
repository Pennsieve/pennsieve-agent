package server

import (
	api "github.com/pennsieve/pennsieve-agent/api/v1"
)

func (s *server) GetTimeseriesRangeForChannels(req *api.GetTimeseriesRangeRequest, stream api.Agent_GetTimeseriesRangeForChannelsServer) error {

    return nil
    //log.Info("Received range request from ID: ", request.Id)
    //
    //fin := make(chan bool)
    //// Save the subscriber stream according to the given client ID
    //s.subscribers.Store(request.Id, sub{stream: stream, finished: fin})
    //
    //ctx := stream.Context()
    //// Keep this scope alive because once this scope exits - the stream is closed
    //for {
    //	select {
    //	case <-fin:
    //		log.Info(fmt.Sprintf("Closing stream for client ID: %d", request.Id))
    //		s.messageSubscribers(fmt.Sprintf("Closing stream for client ID: %d", request.Id))
    //		return nil
    //	case <-ctx.Done():
    //		log.Info(fmt.Sprintf("Client ID %d has disconnected", request.Id))
    //		s.messageSubscribers(fmt.Sprintf("Closing stream for client ID: %d", request.Id))
    //		return nil
    //	}
    //}
}
