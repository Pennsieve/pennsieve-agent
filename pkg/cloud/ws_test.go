package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConn is an in-memory WSConn used by tests. Reads pop from a
// channel; writes append to a buffer guarded by a mutex.
type fakeConn struct {
	reads     chan []byte
	closeOnce sync.Once
	closed    chan struct{}

	mu     sync.Mutex
	writes [][]byte
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		reads:  make(chan []byte, 16),
		closed: make(chan struct{}),
	}
}

func (c *fakeConn) Read(ctx context.Context) ([]byte, error) {
	select {
	case b, ok := <-c.reads:
		if !ok {
			return nil, io.EOF
		}
		return b, nil
	case <-c.closed:
		return nil, io.EOF
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *fakeConn) Write(_ context.Context, data []byte) error {
	select {
	case <-c.closed:
		return io.ErrClosedPipe
	default:
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes = append(c.writes, append([]byte(nil), data...))
	return nil
}

func (c *fakeConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

func (c *fakeConn) deliver(b []byte) {
	select {
	case c.reads <- b:
	case <-c.closed:
	}
}

func (c *fakeConn) writtenEnvelopes() []Envelope {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Envelope, 0, len(c.writes))
	for _, w := range c.writes {
		var env Envelope
		if err := json.Unmarshal(w, &env); err == nil {
			out = append(out, env)
		}
	}
	return out
}

// stubDialer / stubTokenSource used to drive the WS client without a
// real network.

type stubDialer struct {
	conns []WSConn
	idx   int
	err   error
	mu    sync.Mutex
}

func (s *stubDialer) Dial(_ context.Context, _, _ string) (WSConn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	if s.idx >= len(s.conns) {
		return nil, errors.New("stubDialer: no more connections")
	}
	c := s.conns[s.idx]
	s.idx++
	return c, nil
}

type stubTokenSource struct {
	token string
	err   error
}

func (s *stubTokenSource) Token(_ context.Context) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.token, nil
}

func TestWSClient_DispatchesCommandToCallback(t *testing.T) {
	conn := newFakeConn()
	received := make(chan Envelope, 1)

	cli := NewWSClient(WSClientConfig{
		WSURL:     "wss://x",
		Heartbeat: time.Hour, // disable for this test
		Dialer:    &stubDialer{conns: []WSConn{conn}},
		TokenSource: &stubTokenSource{token: "tok"},
		OnCommand: func(_ context.Context, env Envelope) {
			received <- env
		},
	})
	require.NoError(t, cli.Start(context.Background()))
	t.Cleanup(cli.Stop)

	cmd, _ := json.Marshal(Envelope{MessageId: "c1", Kind: KindCommand, Type: "ping"})
	conn.deliver(cmd)

	select {
	case env := <-received:
		assert.Equal(t, "c1", env.MessageId)
		assert.Equal(t, "ping", env.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("OnCommand not invoked")
	}
}

func TestWSClient_SendIsWritten(t *testing.T) {
	conn := newFakeConn()
	cli := NewWSClient(WSClientConfig{
		WSURL:       "wss://x",
		Heartbeat:   time.Hour,
		Dialer:      &stubDialer{conns: []WSConn{conn}},
		TokenSource: &stubTokenSource{token: "tok"},
	})
	require.NoError(t, cli.Start(context.Background()))
	t.Cleanup(cli.Stop)

	require.NoError(t, cli.Send(context.Background(), Envelope{
		MessageId: "ev-1", Kind: KindEvent, Type: "x",
	}))

	// Wait briefly for the write to land.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(conn.writtenEnvelopes()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	envs := conn.writtenEnvelopes()
	require.NotEmpty(t, envs)
	assert.Equal(t, "ev-1", envs[0].MessageId)
}

func TestWSClient_ReconnectsAfterDial(t *testing.T) {
	c1 := newFakeConn()
	c2 := newFakeConn()
	dialer := &stubDialer{conns: []WSConn{c1, c2}}
	cli := NewWSClient(WSClientConfig{
		WSURL:       "wss://x",
		Heartbeat:   time.Hour,
		Dialer:      dialer,
		TokenSource: &stubTokenSource{token: "tok"},
	})
	require.NoError(t, cli.Start(context.Background()))
	t.Cleanup(cli.Stop)

	// Wait for first dial to take effect, then close it to force a
	// reconnect.
	deadline := time.Now().Add(2 * time.Second)
	for dialer.idx < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	_ = c1.Close()

	// Verify the dialer was called a second time.
	deadline = time.Now().Add(3 * time.Second)
	for dialer.idx < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	assert.GreaterOrEqual(t, dialer.idx, 2, "client must redial after disconnect")
}

func TestNextBackoff_Caps(t *testing.T) {
	cur := time.Second
	for range 10 {
		cur = nextBackoff(cur)
	}
	assert.LessOrEqual(t, cur, 60*time.Second)
}
