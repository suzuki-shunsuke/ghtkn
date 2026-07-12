package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

const (
	// defaultRefreshTokenTTL is how long a stored token may sit unused before the sweep
	// discards it, when the unlock request does not specify one. One week balances
	// convenience against how long an unused refresh token lingers.
	defaultRefreshTokenTTL = 7 * 24 * time.Hour
	// refreshTokenSweepInterval is how often the sweep runs while the agent is unlocked.
	// Checking every stored token's expiration daily is cheap relative to the risk of an
	// unused refresh token lingering.
	refreshTokenSweepInterval = 24 * time.Hour
)

// startRefreshTokenSweep launches the background job that discards tokens unused for
// longer than ttl. It runs once immediately and then every refreshTokenSweepInterval
// until ctx is canceled (agent shutdown). It is started only when refresh tokens are
// enabled, since that is when an unused refresh token is worth reclaiming; with refresh
// disabled, unlock strips refresh tokens outright instead.
//
// It is called from handleUnlock with c.mu held; it spawns a goroutine and returns. The
// store is passed in directly so the goroutine does not depend on the locked state.
func (c *Controller) startRefreshTokenSweep(ctx context.Context, st *tokenstore.Store, ttl time.Duration) {
	go func() {
		c.sweepExpiredTokens(st, ttl)
		ticker := time.NewTicker(refreshTokenSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.sweepExpiredTokens(st, ttl)
			}
		}
	}()
}

// sweepExpiredTokens deletes every stored token whose access token expired more than
// ttl ago. A token file is rewritten with a fresh expiration each time it is minted or
// refreshed, so an expiration far in the past means the token has not been used for at
// least ttl; discarding the whole file reclaims the lingering refresh token and keeps
// stale files from accumulating. It is best-effort: read/delete errors are logged and
// skipped.
func (c *Controller) sweepExpiredTokens(st *tokenstore.Store, ttl time.Duration) {
	ids, err := st.ClientIDs()
	if err != nil {
		if c.logger != nil {
			slogerr.WithError(c.logger, err).Warn("list stored tokens for the refresh-token sweep")
		}
		return
	}
	cutoff := c.now().Add(-ttl)
	for _, id := range ids {
		raw, ok, err := st.Get(id)
		if err != nil || !ok {
			continue
		}
		expired := tokenExpiredBefore(raw, cutoff)
		scrub(raw)
		if !expired {
			continue
		}
		if err := st.Delete(id); err != nil {
			if c.logger != nil {
				slogerr.WithError(c.logger, err).Warn("discard an unused token in the refresh-token sweep", "client_id", id)
			}
			continue
		}
		if c.logger != nil {
			c.logger.Info("discarded a token unused past the refresh TTL", "client_id", id)
		}
	}
}

// tokenExpiredBefore reports whether the token's access-token expiration is before
// cutoff. An unparseable token, or one without an expiration, is treated as not expired
// so a decode glitch never deletes data.
func tokenExpiredBefore(raw json.RawMessage, cutoff time.Time) bool {
	// Only the expiration is needed; do not materialize the tokens as Go strings.
	token := &struct {
		ExpirationDate time.Time `json:"expiration_date"`
	}{}
	if err := json.Unmarshal(raw, token); err != nil {
		return false
	}
	if token.ExpirationDate.IsZero() {
		return false
	}
	return token.ExpirationDate.Before(cutoff)
}
