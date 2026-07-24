package agent

import (
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// handleStatus reports whether the agent is locked, how many tokens are cached
// (when unlocked), whether an agent key already exists on disk, and whether refresh
// tokens are enabled.
func (c *Controller) handleStatus() *agentapi.Response {
	st := c.tokenStore()
	enabled, ttl := c.refreshState()
	resp := &agentapi.Response{OK: true, Locked: st == nil, Initialized: c.keyExists(), RefreshTokenEnabled: enabled}
	if st != nil {
		resp.Count = st.Len()
	}
	if enabled {
		// The TTL is part of the unlocked, refresh-enabled state, so report it only then
		// (a locked agent has refresh off). A client such as `ghtkn info` shows it.
		resp.RefreshTokenTTL = ttl
	}
	return resp
}
