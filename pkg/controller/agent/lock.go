package agent

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
func (c *Controller) handleLock() *agentapi.Response {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.store == nil {
		return &agentapi.Response{OK: true, Locked: true}
	}
	if c.sweepCancel != nil {
		// Stop the sweep bound to this unlock before scrubbing the key it uses.
		c.sweepCancel()
		c.sweepCancel = nil
	}
	c.store.Zero()
	c.store = nil
	c.enableRefreshToken = false
	c.refreshTokenTTL = 0
	if c.logger != nil {
		c.logger.Info("agent locked")
	}
	return &agentapi.Response{OK: true, Locked: true}
}
