package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/refreshtoken"
	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/tokenstore"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

const (
	// defaultRefreshTokenTTL is how long a stored token may sit unused before the sweep
	// discards it, when the unlock request does not specify one. Three days keeps the
	// window in which a rarely used app's refresh token lingers (and can be minted from)
	// short, while leaving apps used every few days untouched; use --refresh-token-ttl to
	// trade convenience for a longer window.
	defaultRefreshTokenTTL = 3 * 24 * time.Hour
	// refreshTokenSweepInterval is how often the sweep runs while the agent is unlocked.
	// Checking every stored token's expiration daily is cheap relative to the risk of an
	// unused refresh token lingering.
	refreshTokenSweepInterval = 24 * time.Hour
)

// resolveRefreshTokenTTL clamps a requested refresh-token TTL to a sane range: a
// non-positive value falls back to the default, and a value above the six-month maximum
// is capped. The CLI rejects an over-large TTL up front, so the cap is a server-side
// backstop for any other client rather than a normal path.
func (s *Server) resolveRefreshTokenTTL(ttl time.Duration) time.Duration {
	switch {
	case ttl <= 0:
		return defaultRefreshTokenTTL
	case ttl > refreshtoken.MaxTTL:
		if s.logger != nil {
			s.logger.Warn("refresh-token-ttl exceeds the maximum; capping it", "requested", ttl, "max", refreshtoken.MaxTTL)
		}
		return refreshtoken.MaxTTL
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
func (s *Server) startRefreshTokenSweep(ctx context.Context, st *tokenstore.Store, ttl time.Duration) {
	go func() {
		s.sweepExpiredTokens(st, ttl)
		ticker := time.NewTicker(refreshTokenSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.sweepExpiredTokens(st, ttl)
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
func (s *Server) sweepExpiredTokens(st *tokenstore.Store, ttl time.Duration) {
	ids, err := st.ClientIDs()
	if err != nil {
		if s.logger != nil {
			slogerr.WithError(s.logger, err).Warn("list stored tokens for the refresh-token sweep")
		}
		return
	}
	cutoff := time.Now().Add(-ttl)
	for _, id := range ids {
		// Read the expiration and delete under the store's lock in one operation, so a
		// concurrent refresh that stores a fresh token cannot slip in between the check and
		// the delete and have its new token discarded (see Store.DeleteIf).
		deleted, err := st.DeleteIf(id, func(raw json.RawMessage) bool {
			return tokenExpiredBefore(raw, cutoff)
		})
		if err != nil {
			if s.logger != nil {
				slogerr.WithError(s.logger, err).Warn("discard an unused token in the refresh-token sweep", "client_id", id)
			}
			continue
		}
		if deleted && s.logger != nil {
			s.logger.Info("discarded a token unused past the refresh TTL", "client_id", id)
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
