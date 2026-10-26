package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// WSConn abstracts the underlying WebSocket implementation. The
// production implementation wraps coder/websocket; tests provide a
// fake. Methods are expected to be safe for concurrent use by separate
// reader/writer goroutines (one Read at a time, one Write at a time).
type WSConn interface {
	Read(ctx context.Context) ([]byte, error)
	Write(ctx context.Context, data []byte) error
	Close() error
}

// Dialer dials the given ws url with the given short-lived token.
type Dialer interface {
	Dial(ctx context.Context, wsURL, token string) (WSConn, error)
}

// TokenSource exchanges the long-lived agentSecret for a short-lived
// ws token. In production this calls the REST /agents/ws-token; tests
// stub it with a static token.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// CommandCallback is invoked for every inbound Kind=command envelope.
type CommandCallback func(ctx context.Context, env Envelope)

// WSClientConfig configures a WSClient. Defaults are set in
// NewWSClient if zero values are present.
type WSClientConfig struct {
	BaseRESTURL string
	WSURL       string
	AgentId     string
	AgentSecret string
	Heartbeat   time.Duration
	OnCommand   CommandCallback

	// Optional overrides — used by tests.
	Dialer      Dialer
	TokenSource TokenSource
}

// WSClient maintains a single WebSocket connection. On disconnect it
// reconnects with exponential backoff. Outbound writes are serialized
// through a single goroutine; inbound reads are dispatched to
// OnCommand for command envelopes and dropped for everything else.
type WSClient struct {
	cfg WSClientConfig

	dialer      Dialer
	tokenSource TokenSource

	mu       sync.Mutex
	conn     WSConn
	stopChan chan struct{}
	stopped  bool

	send chan Envelope
}

func NewWSClient(cfg WSClientConfig) *WSClient {
	if cfg.Heartbeat == 0 {
		cfg.Heartbeat = 270 * time.Second
	}
	w := &WSClient{
		cfg:      cfg,
		stopChan: make(chan struct{}),
		send:     make(chan Envelope, 32),
	}
	w.dialer = cfg.Dialer
	if w.dialer == nil {
		w.dialer = defaultDialer{}
	}
	w.tokenSource = cfg.TokenSource
	if w.tokenSource == nil {
		w.tokenSource = &restTokenSource{
			client:      NewClient(cfg.BaseRESTURL, ""),
			agentId:     cfg.AgentId,
			agentSecret: cfg.AgentSecret,
		}
	}
	return w
}

// Start launches the connect/reconnect loop. Returns immediately.
func (w *WSClient) Start(ctx context.Context) error {
	go w.run(ctx)
	return nil
}

func (w *WSClient) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	close(w.stopChan)
	if w.conn != nil {
		_ = w.conn.Close()
	}
	w.mu.Unlock()
}

// Send enqueues an envelope for delivery on the next available
// connection. Caller is responsible for setting MessageId / Kind / etc.
func (w *WSClient) Send(ctx context.Context, env Envelope) error {
	select {
	case w.send <- env:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-w.stopChan:
		return fmt.Errorf("ws client stopped")
	}
}

func (w *WSClient) run(ctx context.Context) {
	backoff := time.Second
	for {
		select {
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		default:
		}

		token, err := w.tokenSource.Token(ctx)
		if err != nil {
			waitOrStop(w.stopChan, backoff)
			backoff = nextBackoff(backoff)
			continue
		}
		conn, err := w.dialer.Dial(ctx, w.cfg.WSURL, token)
		if err != nil {
			waitOrStop(w.stopChan, backoff)
			backoff = nextBackoff(backoff)
			continue
		}
		backoff = time.Second

		w.mu.Lock()
		w.conn = conn
		w.mu.Unlock()

		w.serve(ctx, conn)

		_ = conn.Close()
		w.mu.Lock()
		w.conn = nil
		w.mu.Unlock()
	}
}

func (w *WSClient) serve(ctx context.Context, conn WSConn) {
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			b, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var env Envelope
			if err := json.Unmarshal(b, &env); err != nil {
				continue
			}
			if env.Kind == KindCommand && w.cfg.OnCommand != nil {
				w.cfg.OnCommand(ctx, env)
			}
		}
	}()

	heartbeat := time.NewTicker(w.cfg.Heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		case <-readDone:
			return
		case env := <-w.send:
			b, err := json.Marshal(env)
			if err != nil {
				continue
			}
			if err := conn.Write(ctx, b); err != nil {
				return
			}
		case <-heartbeat.C:
			b, _ := json.Marshal(Envelope{Kind: KindHeartbeat})
			if err := conn.Write(ctx, b); err != nil {
				return
			}
		}
	}
}

func waitOrStop(stop <-chan struct{}, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-stop:
	case <-t.C:
	}
}

func nextBackoff(cur time.Duration) time.Duration {
	next := cur * 2
	if next > 60*time.Second {
		next = 60 * time.Second
	}
	return next
}

// restTokenSource is the production TokenSource — exchanges
// agentSecret for a fresh ws token via /agents/ws-token.
type restTokenSource struct {
	client      *Client
	agentId     string
	agentSecret string
}

func (r *restTokenSource) Token(ctx context.Context) (string, error) {
	resp, err := r.client.WSToken(ctx, WSTokenRequest{
		AgentId:     r.agentId,
		AgentSecret: r.agentSecret,
	})
	if err != nil {
		return "", err
	}
	return resp.Token, nil
}
