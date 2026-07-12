package agent

import (
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// handleStatus reports whether the agent is locked, how many tokens are cached
// (when unlocked), whether an agent key already exists on disk, and whether refresh
// tokens are enabled.
func (c *Controller) handleStatus() *agentapi.Response {
	st := c.tokenStore()
	resp := &agentapi.Response{OK: true, Locked: st == nil, Initialized: c.keyExists(), RefreshTokenEnabled: c.refreshEnabled()}
	if st != nil {
		resp.Count = st.Len()
	}
	return resp
}
