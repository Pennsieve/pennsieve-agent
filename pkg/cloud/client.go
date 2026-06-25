package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a thin REST client for the Pennsieve Agent Service. It is
// deliberately minimal: registration, ws-token exchange, and
// reconciliation. WebSocket transport is in ws.go.
type Client struct {
	BaseURL string
	HTTP    *http.Client

	// JWT is the user's Pennsieve session token, used for endpoints
	// that require user authentication (registration, reconcile).
	// Empty when calling unauthenticated endpoints (ws-token).
	JWT string
}

func NewClient(baseURL, jwt string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		JWT:     jwt,
	}
}

func (c *Client) Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	var out RegisterResponse
	err := c.doJSON(ctx, http.MethodPost, "/agents/register", req, &out, true)
	return out, err
}

func (c *Client) WSToken(ctx context.Context, req WSTokenRequest) (WSTokenResponse, error) {
	var out WSTokenResponse
	err := c.doJSON(ctx, http.MethodPost, "/agents/ws-token", req, &out, false)
	return out, err
}

type ReconcileCommandView struct {
	CommandId string `json:"commandId"`
	Status    string `json:"status"`
}

type ReconcileRequest struct {
	AgentId  string                 `json:"agentId"`
	Commands []ReconcileCommandView `json:"commands"`
}

type ReconcileResponse struct {
	Outstanding []OutstandingCommand `json:"outstanding"`
}

type OutstandingCommand struct {
	CommandId string          `json:"commandId"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

func (c *Client) Reconcile(ctx context.Context, agentId string, req ReconcileRequest) (ReconcileResponse, error) {
	var out ReconcileResponse
	err := c.doJSON(ctx, http.MethodPost, "/agents/"+agentId+"/reconcile", req, &out, true)
	return out, err
}

func (c *Client) doJSON(ctx context.Context, method, path string, in, out any, authed bool) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if authed {
		if c.JWT == "" {
			return fmt.Errorf("cloud: %s requires JWT", path)
		}
		req.Header.Set("Authorization", "Bearer "+c.JWT)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloud: %s %s: %s: %s", method, path, resp.Status, string(b))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
