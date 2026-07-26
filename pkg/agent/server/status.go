package server

import (
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// handleStatus reports whether the agent is locked, how many tokens are cached
// (when unlocked), whether an agent key already exists on disk, whether refresh
// tokens are enabled, and which ghtkn version and protocol versions the agent speaks.
//
// The version fields describe the agent process itself, not its unlocked state, so they
// are reported even while locked: a client comparing its own version with the running
// agent's must not have to unlock it first. The newest protocol version is stamped on
// every response (see serve), so only the minimum is set here; it is meaningful only
// alongside the maximum, which STATUS carries anyway.
func (s *Server) handleStatus() *agentapi.Response {
	st := s.tokenStore()
	enabled, ttl := s.refreshState()
	resp := &agentapi.Response{
		OK: true, Locked: st == nil, Initialized: s.keyExists(), RefreshTokenEnabled: enabled,
		Version: s.version, MinProtocolVersion: agentapi.MinProtocolVersion,
	}
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
