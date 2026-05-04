package server

import (
	"context"
	"time"

	"github.com/pennsieve/pennsieve-agent/v2/pkg/reconciler"
)

// reconcilerInterval is the period between reconciliation passes. Chosen
// loose because async finalize-import lag is typically seconds and a stale
// minute is fine; smaller values just burn API calls on idle agents.
const reconcilerInterval = 60 * time.Second

// StartReconciler launches the local-state reconciler on the given context
// and blocks until the context is cancelled. Intended to be invoked in its
// own goroutine from the agent's startup path so its lifetime matches the
// gRPC server's.
func (s *agentServer) StartReconciler(ctx context.Context) {
	r := reconciler.New(
		s.ManifestStore(),
		s.ManifestFileStore(),
		s.PennsieveClient,
		reconcilerInterval,
	)
	r.Run(ctx)
}