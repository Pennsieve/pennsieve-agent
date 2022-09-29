package shared

import (
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HandleAgentError Outputs messages to users in response to GRPC errors.
func HandleAgentError(err error, defaultMessage string) {
	if err != nil {

		st := status.Convert(err)
		switch st.Code() {
		case codes.Unavailable:
			fmt.Println(`Error: Unable to connect to Pennsieve Agent.

Please restart the agent using 'pennsieve agent' command.`)
		default:
			fmt.Println(defaultMessage)
		}
		return
	}
}
