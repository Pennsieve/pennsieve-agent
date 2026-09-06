package cloud

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// Manager orchestrates registration with the Pennsieve Agent Service
// and maintains a WebSocket connection for the active profile.
//
// Workflow:
//   - On Start: ensure local installation id exists, register the
//     active profile (mints agentId+agentSecret on first run, refreshes
//     on subsequent runs), then connect the WS client.
//   - On profile switch: stop the WS client, register the new profile,
//     start a fresh WS client.
//
// The agent can talk to the platform via SendEvent (out-of-band agent->
// server messages) and respond to inbound commands by registering
// handlers on the Manager.
type Manager struct {
	store   *Store
	rest    *Client
	wsCli   *WSClient
	profile string
	reg     Registration
	inst    Installation

	hostname string
	osName   string
	version  string

	// Optional overrides — primarily for tests. If nil, the default
	// (production) implementations are used.
	Dialer      Dialer
	TokenSource TokenSource

	handlers map[string]CommandHandler
}

// CommandHandler runs a single command. Implementations should be
// idempotent where possible. Status transitions (RUNNING, COMPLETED,
// FAILED) are recorded by the Manager via the WSClient and SQLite.
type CommandHandler func(ctx context.Context, payload []byte) ([]byte, error)

func NewManager(store *Store, baseURL, jwt, hostname, version string) *Manager {
	return &Manager{
		store:    store,
		rest:     NewClient(baseURL, jwt),
		hostname: hostname,
		osName:   runtime.GOOS,
		version:  version,
		handlers: map[string]CommandHandler{},
	}
}

// RegisterHandler binds a CommandHandler to a server command type.
// Unhandled types are reported as failures back to the service.
func (m *Manager) RegisterHandler(cmdType string, h CommandHandler) {
	m.handlers[cmdType] = h
}

// Start performs the full bootstrap for the given profile: identity,
// registration, and WebSocket attach.
func (m *Manager) Start(ctx context.Context, profile string) error {
	m.profile = profile

	inst, err := m.store.GetOrCreateInstallation(func() string { return uuid.New().String() })
	if err != nil {
		return fmt.Errorf("cloud: get installation: %w", err)
	}
	m.inst = inst

	reg, err := m.ensureRegistered(ctx)
	if err != nil {
		return err
	}
	m.reg = reg

	m.wsCli = NewWSClient(WSClientConfig{
		BaseRESTURL: m.rest.BaseURL,
		WSURL:       reg.WSURL,
		AgentId:     reg.AgentId,
		AgentSecret: reg.AgentSecret,
		Heartbeat:   time.Duration(reg.HeartbeatSec) * time.Second,
		OnCommand:   m.handleCommand,
		Dialer:      m.Dialer,
		TokenSource: m.TokenSource,
	})
	return m.wsCli.Start(ctx)
}

// Stop tears down the WS connection. The local SQLite registration is
// left intact so a later Start resumes the same agentId.
func (m *Manager) Stop() {
	if m.wsCli != nil {
		m.wsCli.Stop()
	}
}

func (m *Manager) AgentId() string { return m.reg.AgentId }

// SendEvent emits an agent->server event over the WS.
func (m *Manager) SendEvent(ctx context.Context, eventType string, payload []byte) error {
	if m.wsCli == nil {
		return fmt.Errorf("cloud: not connected")
	}
	return m.wsCli.Send(ctx, Envelope{
		MessageId: uuid.New().String(),
		Kind:      KindEvent,
		Type:      eventType,
		Payload:   payload,
	})
}

func (m *Manager) ensureRegistered(ctx context.Context) (Registration, error) {
	existing, err := m.store.GetRegistration(m.profile)
	if err != nil && err != ErrNotFound {
		return Registration{}, err
	}

	req := RegisterRequest{
		InstallationId: m.inst.InstallationId,
		Hostname:       m.hostname,
		OS:             m.osName,
		AgentVersion:   m.version,
		ProfileName:    m.profile,
	}
	if existing.AgentId != "" {
		req.AgentId = existing.AgentId
	}

	resp, err := m.rest.Register(ctx, req)
	if err != nil {
		return Registration{}, fmt.Errorf("cloud: register: %w", err)
	}

	reg := Registration{
		ProfileName:  m.profile,
		AgentId:      resp.AgentId,
		AgentSecret:  existing.AgentSecret,
		WSURL:        resp.WSURL,
		WSTokenURL:   resp.WSTokenURL,
		HeartbeatSec: resp.HeartbeatSec,
		RegisteredAt: time.Now(),
	}
	// Server only returns the agent secret on first registration (or
	// rotation) — preserve the previous value otherwise.
	if resp.AgentSecret != "" {
		reg.AgentSecret = resp.AgentSecret
	}
	if reg.AgentSecret == "" {
		return Registration{}, fmt.Errorf("cloud: missing agent secret after registration")
	}

	if err := m.store.UpsertRegistration(reg); err != nil {
		return Registration{}, fmt.Errorf("cloud: persist registration: %w", err)
	}
	return reg, nil
}

func (m *Manager) handleCommand(ctx context.Context, env Envelope) {
	now := time.Now()

	// Idempotency: skip if we have already seen this commandId.
	seen, err := m.store.SeenCommand(env.MessageId)
	if err == nil && seen {
		_ = m.wsCli.Send(ctx, Envelope{
			MessageId: env.MessageId,
			Kind:      KindAck,
			Payload:   marshalJSON(CommandAck{CommandId: env.MessageId}),
		})
		return
	}

	_ = m.store.UpsertCommand(LocalCommand{
		CommandId:  env.MessageId,
		Type:       env.Type,
		Payload:    env.Payload,
		Status:     CmdStatusReceived,
		ReceivedAt: now,
	})
	_ = m.wsCli.Send(ctx, Envelope{
		MessageId: env.MessageId,
		Kind:      KindAck,
		Payload:   marshalJSON(CommandAck{CommandId: env.MessageId}),
	})

	handler, ok := m.handlers[env.Type]
	if !ok {
		m.reportResult(ctx, env.MessageId, CmdStatusFailed, nil, "no handler registered for type "+env.Type)
		return
	}

	started := time.Now()
	_ = m.store.UpsertCommand(LocalCommand{
		CommandId: env.MessageId,
		Type:      env.Type,
		Payload:   env.Payload,
		Status:    CmdStatusRunning,
		StartedAt: &started,
	})

	result, err := handler(ctx, env.Payload)
	if err != nil {
		m.reportResult(ctx, env.MessageId, CmdStatusFailed, nil, err.Error())
		return
	}
	m.reportResult(ctx, env.MessageId, CmdStatusCompleted, result, "")
}

func (m *Manager) reportResult(ctx context.Context, commandId, status string, result []byte, errMsg string) {
	completed := time.Now()
	_ = m.store.UpsertCommand(LocalCommand{
		CommandId:   commandId,
		Status:      status,
		CompletedAt: &completed,
		Result:      result,
		Error:       errMsg,
	})
	res := CommandResult{
		CommandId: commandId,
		Status:    status,
		Result:    result,
		Error:     errMsg,
	}
	_ = m.wsCli.Send(ctx, Envelope{
		MessageId: commandId,
		Kind:      KindResult,
		Payload:   marshalJSON(res),
	})
}

func marshalJSON(v any) []byte {
	b, _ := jsonMarshal(v)
	return b
}
