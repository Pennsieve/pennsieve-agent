package cloud

import "encoding/json"

// Wire envelope shared with pennsieve-agent-service. Keep this in sync
// with internal/models/wire.go in the service repo.
type Envelope struct {
	MessageId string          `json:"id"`
	Kind      string          `json:"kind"`
	Type      string          `json:"type,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

const (
	KindEvent     = "event"
	KindCommand   = "command"
	KindAck       = "ack"
	KindResult    = "result"
	KindHeartbeat = "heartbeat"
	KindReconcile = "reconcile"
)

const (
	CmdStatusReceived  = "DELIVERED"
	CmdStatusRunning   = "RUNNING"
	CmdStatusCompleted = "COMPLETED"
	CmdStatusFailed    = "FAILED"
)

type CommandAck struct {
	CommandId string `json:"commandId"`
}

type CommandResult struct {
	CommandId string          `json:"commandId"`
	Status    string          `json:"status"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// REST DTOs (mirror service models).
type RegisterRequest struct {
	InstallationId string `json:"installationId"`
	AgentId        string `json:"agentId,omitempty"`
	Hostname       string `json:"hostname"`
	OS             string `json:"os"`
	AgentVersion   string `json:"agentVersion"`
	ProfileName    string `json:"profileName"`
}

type RegisterResponse struct {
	AgentId      string `json:"agentId"`
	AgentSecret  string `json:"agentSecret,omitempty"`
	WSURL        string `json:"wsUrl"`
	WSTokenURL   string `json:"wsTokenUrl"`
	HeartbeatSec int    `json:"heartbeatSec"`
}

type WSTokenRequest struct {
	AgentId     string `json:"agentId"`
	AgentSecret string `json:"agentSecret"`
}

type WSTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
}
