package agent

import (
	"errors"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/keyfile"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
)

// handleUnlock loads (or creates) the data key from the request passphrase and
// switches the agent to an unlocked, disk-backed store. It is idempotent: unlocking
// an already-unlocked agent succeeds without re-reading the key.
func (c *Controller) handleUnlock(req *agentapi.Request) *agentapi.Response {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.store != nil {
		// Already unlocked: the refresh setting is fixed at the first unlock and can't
		// be flipped here (this path never verifies the passphrase). Report the current
		// state so a re-unlock still shows it.
		return &agentapi.Response{OK: true, RefreshTokenEnabled: c.enableRefreshToken}
	}
	dataKey, created, err := keyfile.LoadOrCreateDataKey(c.keyFile, []byte(req.Passphrase))
	if err != nil {
		if errors.Is(err, keyfile.ErrIncorrectPassphrase) {
			return &agentapi.Response{Error: keyfile.ErrIncorrectPassphrase.Error()}
		}
		return &agentapi.Response{Error: errMsgUnlock}
	}
	c.store = tokenstore.New(dataKey, c.tokenDir)
	// Bind refresh enablement to this passphrase-authenticated unlock.
	c.enableRefreshToken = req.EnableRefreshToken
	if c.logger != nil {
		if created {
			c.logger.Info("generated a new agent key", "path", c.keyFile)
			// A new key can't decrypt token files written under a previous key
			// (e.g. when the key file was deleted while the tokens remained), so
			// warn that those cached tokens are orphaned and will be re-minted.
			if n := c.store.Len(); n > 0 {
				c.logger.Warn("found cached token files that predate the new agent key; they can't be decrypted and will be re-minted on the next get", "path", c.tokenDir, "count", n)
			}
		}
		c.logger.Info("agent unlocked", "refresh_token_enabled", c.enableRefreshToken)
	}
	return &agentapi.Response{OK: true, RefreshTokenEnabled: c.enableRefreshToken}
}
