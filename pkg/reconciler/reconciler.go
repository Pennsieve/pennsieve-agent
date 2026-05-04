// Package reconciler converges the agent's local manifest-file status with
// the server's canonical status by polling.
//
// The two-phase finalize endpoint on upload-service-v2 returns "finalized"
// once a file is enqueued for import (SQS), not once the upload lambda has
// actually written the Postgres row and flipped DynamoDB to Finalized. The
// agent therefore has two ways its local view can drift from server reality:
//
//  1. Agent calls finalize, server enqueues, agent locally marks Finalized.
//     Upload lambda then fails import-with-retries and the file ends up in
//     the DLQ. Server still shows it pre-Finalized; agent shows Finalized.
//
//  2. Agent's finalize call fails entirely (network, server 5xx, batch
//     rejected on validation). Files stay locally Uploaded. Some of those
//     may eventually be imported through other paths or operator action.
//
// The reconciler addresses both by periodically asking the server which
// uploadIds it has at Finalized status, and flipping matching local rows to
// Verified. It runs on a long-lived context tied to the agent daemon's
// lifetime, not to any one upload-session's gRPC handler.
//
// Slated for removal once the server→agent websocket lands; the websocket
// consumer will write the same transitions in response to push events.
package reconciler

import (
	"context"
	"time"

	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/manifest/manifestFile"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	log "github.com/sirupsen/logrus"
)

// ClientProvider returns the current Pennsieve client. Called per pass so a
// reconciler started before the user has logged in still recovers once a
// profile is configured.
type ClientProvider func() (*pennsieve.Client, error)

type Reconciler struct {
	manifestStore     store.ManifestStore
	manifestFileStore store.ManifestFileStore
	clientProvider    ClientProvider
	interval          time.Duration
}

func New(
	manifestStore store.ManifestStore,
	manifestFileStore store.ManifestFileStore,
	clientProvider ClientProvider,
	interval time.Duration,
) *Reconciler {
	return &Reconciler{
		manifestStore:     manifestStore,
		manifestFileStore: manifestFileStore,
		clientProvider:    clientProvider,
		interval:          interval,
	}
}

// Run blocks until ctx is cancelled. Performs an immediate reconciliation
// pass on entry — catches state from previous agent sessions where uploads
// completed but the agent shut down before the server-side import finished —
// and then runs every interval thereafter.
func (r *Reconciler) Run(ctx context.Context) {
	log.Infof("reconciler: starting (interval=%s)", r.interval)
	r.reconcileOnce(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("reconciler: stopping")
			return
		case <-ticker.C:
			r.reconcileOnce(ctx)
		}
	}
}

func (r *Reconciler) reconcileOnce(ctx context.Context) {
	client, err := r.clientProvider()
	if err != nil {
		log.Debugf("reconciler: pennsieve client unavailable, skipping pass: %v", err)
		return
	}

	pendingStatuses := []manifestFile.Status{manifestFile.Uploaded, manifestFile.Finalized}
	ids, err := r.manifestFileStore.GetManifestIDsWithFilesInStatus(pendingStatuses)
	if err != nil {
		log.Errorf("reconciler: list manifests with pending files: %v", err)
		return
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		if err := r.reconcileManifest(ctx, client, id); err != nil {
			log.Errorf("reconciler: manifest %d: %v", id, err)
		}
	}
}

func (r *Reconciler) reconcileManifest(ctx context.Context, client *pennsieve.Client, manifestID int32) error {
	m, err := r.manifestStore.Get(manifestID)
	if err != nil {
		return err
	}
	if !m.NodeId.Valid || m.NodeId.String == "" {
		// Manifest never registered with the server — no canonical state to
		// reconcile against.
		return nil
	}
	// Skip manifests from a different org than the active profile. They're
	// leftovers from a prior session whose credentials this profile no
	// longer has, and querying them just produces 403 noise.
	activeOrg := client.OrganizationNodeId
	if activeOrg != "" && m.OrganizationId != "" && m.OrganizationId != activeOrg {
		log.Debugf("reconciler: manifest %d belongs to org %s (active: %s) — skipping", manifestID, m.OrganizationId, activeOrg)
		return nil
	}
	nodeID := m.NodeId.String

	var continuationToken string
	var verified int
	for {
		resp, err := client.Manifest.GetFilesForStatus(
			ctx, nodeID, manifestFile.Finalized, continuationToken, true,
		)
		if err != nil {
			return err
		}
		if len(resp.Files) > 0 {
			if err := r.manifestFileStore.BatchSetStatus(manifestFile.Verified, resp.Files); err != nil {
				return err
			}
			verified += len(resp.Files)
		}
		if resp.ContinuationToken == "" {
			break
		}
		continuationToken = resp.ContinuationToken
	}

	if verified > 0 {
		log.Infof("reconciler: manifest %s: verified %d file(s)", nodeID, verified)
	}
	return nil
}