package server

import (
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// handleLock discards the in-memory data key and returns the agent to the locked state
// without stopping the process, socket, or key file. It needs no passphrase: it only
// reduces access (UNLOCK re-derives the same data key from the key file, so tokens stored
// before the lock stay readable after the next unlock). Locking an already-locked agent
// is a no-op success.
//
// It stops the refresh-token sweep started at unlock and scrubs the data key. Scrubbing
// runs under the store lock (Store.Zero), so it serializes with an in-flight
// Get/Set/Delete rather than racing it; a request that reads the store after the key is
// zeroed simply fails to decrypt and is treated as a cache miss.
func (s *Server) handleLock() *agentapi.Response {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		return &agentapi.Response{OK: true, Locked: true}
	}
	if s.sweepCancel != nil {
		// Stop the sweep bound to this unlock before scrubbing the key it uses.
		s.sweepCancel()
		s.sweepCancel = nil
	}
	s.store.Zero()
	s.store = nil
	s.enableRefreshToken = false
	s.refreshTokenTTL = 0
	if s.logger != nil {
		s.logger.Info("agent locked")
	}
	return &agentapi.Response{OK: true, Locked: true}
}
