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
	// MaxRefreshTokenTTL caps --refresh-token-ttl: a stored token is useless once its
	// refresh token expires (GitHub issues refresh tokens that live about six months),
	// so a larger TTL is clamped to this. A month is counted as 30 days. This is the
	// single source of truth for the upper bound: the server clamps to it (see
	// resolveRefreshTokenTTL) and the CLI rejects larger values up front by referencing
	// this same constant (see pkg/cli/agent).
	MaxRefreshTokenTTL = 6 * 30 * 24 * time.Hour
	// refreshTokenSweepInterval is how often the sweep runs while the agent is unlocked.
	// Checking every stored token's expiration daily is cheap relative to the risk of an
	// unused refresh token lingering.
	refreshTokenSweepInterval = 24 * time.Hour
)

// resolveRefreshTokenTTL clamps a requested refresh-token TTL to a sane range: a
// non-positive value falls back to the default, and a value above the six-month maximum
// is capped. The CLI rejects an over-large TTL up front, so the cap is a server-side
// backstop for any other client rather than a normal path.
func (c *Controller) resolveRefreshTokenTTL(ttl time.Duration) time.Duration {
	switch {
	case ttl <= 0:
		return defaultRefreshTokenTTL
	case ttl > MaxRefreshTokenTTL:
		if c.logger != nil {
			c.logger.Warn("refresh-token-ttl exceeds the maximum; capping it", "requested", ttl, "max", MaxRefreshTokenTTL)
		}
		return MaxRefreshTokenTTL
	default:
		return ttl
	}
}

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
	cutoff := time.Now().Add(-ttl)
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
// cutoff. An unparsable token, or one without an expiration, is treated as not expired
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
