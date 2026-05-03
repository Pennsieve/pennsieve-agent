package cloud

import (
	"context"
	"fmt"
)

// defaultDialer is the production Dialer. It is a thin shim around
// the chosen WebSocket library and is intentionally separated from
// the WSClient so tests can substitute a fake without dragging the
// real network library into the test build.
//
// To keep the dependency footprint of pkg/cloud minimal until the
// websocket library choice is finalized at the project level, this
// default dialer is a placeholder: it returns an error if used. To
// activate WebSocket connectivity, plug in a concrete Dialer at
// Manager construction time (see WSClientConfig.Dialer in ws.go).
//
// The recommended production library is github.com/coder/websocket
// (the maintained successor to nhooyr.io/websocket); a one-file
// adapter that satisfies the WSConn interface is straightforward.
type defaultDialer struct{}

func (defaultDialer) Dial(ctx context.Context, wsURL, token string) (WSConn, error) {
	return nil, fmt.Errorf("cloud: no WebSocket dialer configured; supply WSClientConfig.Dialer")
}
