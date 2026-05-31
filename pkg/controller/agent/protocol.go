package agent

import "encoding/json"

// Command names used in the agent socket protocol.
const (
	CommandGet    = "GET"
	CommandSet    = "SET"
	CommandStatus = "STATUS"
)

// Request is a single request sent by a client to the agent.
// The wire format is one JSON object per line (newline-delimited JSON).
type Request struct {
	// Command is one of CommandGet, CommandSet, or CommandStatus.
	Command string `json:"command"`
	// ClientID identifies the GitHub App (used by GET and SET).
	ClientID string `json:"client_id,omitempty"`
	// Token is the opaque access token payload (used by SET).
	Token json.RawMessage `json:"token,omitempty"`
}

// Response is a single response returned by the agent for a Request.
// The wire format is one JSON object per line (newline-delimited JSON).
type Response struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`
	// Token is the cached access token payload (returned by a successful GET).
	Token json.RawMessage `json:"token,omitempty"`
	// Count is the number of cached tokens (returned by STATUS).
	Count int `json:"count,omitempty"`
	// Error describes the failure when OK is false.
	Error string `json:"error,omitempty"`
}
