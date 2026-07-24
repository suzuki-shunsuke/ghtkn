package server

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/keyfile"
	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/tokenstore"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// handleUnlock loads (or creates) the data key from the request passphrase and
// switches the agent to an unlocked, disk-backed store. It is idempotent: unlocking
// an already-unlocked agent succeeds without re-reading the key.
//
// Refresh-token handling is bound to this passphrase-authenticated unlock. When refresh
// is enabled it starts the periodic sweep (see sweep.go) that discards tokens unused
// past the TTL. When refresh is disabled it strips every stored refresh token, so a
// refresh token left over from a previous refresh-enabled run can no longer leak. ctx is
// the server context; the sweep it starts runs until the agent shuts down.
func (s *Server) handleUnlock(ctx context.Context, req *agentapi.Request) *agentapi.Response {
	// The passphrase is only needed to derive the data key; zero it afterwards. Scrub on
	// entry so it is zeroed even on the already-unlocked early return below.
	defer scrub(req.Passphrase)
	// Refuse rather than silently unlock with refresh off: the client asked for a feature
	// this OS can't keep safe (see RefreshTokenSupported), and leaving the agent locked
	// makes that impossible to miss. The CLI rejects --enable-refresh before prompting
	// for the passphrase; this covers any other client.
	if req.EnableRefreshToken && !RefreshTokenSupported(s.goos) {
		return &agentapi.Response{Error: errMsgRefreshTokenUnsupportedOS}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store != nil {
		// Already unlocked: the refresh setting is fixed at the first unlock and can't
		// be flipped here (this path never verifies the passphrase). Report the current
		// state so a re-unlock still shows it.
		return &agentapi.Response{OK: true, RefreshTokenEnabled: s.enableRefreshToken}
	}
	dataKey, created, err := keyfile.LoadOrCreateDataKey(s.keyFile, req.Passphrase)
	if err != nil {
		if errors.Is(err, keyfile.ErrIncorrectPassphrase) {
			return &agentapi.Response{Error: keyfile.ErrIncorrectPassphrase.Error()}
		}
		return &agentapi.Response{Error: errMsgUnlock}
	}
	store := tokenstore.New(dataKey, s.tokenDir)
	// Refresh is being turned off while a still-valid refresh token is stored: dropping it
	// forces the affected apps back through the device flow, so do not do it silently on a
	// forgotten --enable-refresh. Answer with RefreshTokenRemovalPending (staying locked,
	// nothing bound) so the client can prompt; a confirmed re-unlock carries
	// ConfirmRefreshTokenRemoval and falls through to strip below. A first-ever unlock has
	// no stored tokens, so this never blocks key creation.
	if s.needsRefreshRemovalConfirmation(req, store) {
		if s.logger != nil {
			s.logger.Info("unlock without --enable-refresh found stored refresh tokens; awaiting confirmation to drop them")
		}
		return &agentapi.Response{RefreshTokenRemovalPending: true, Error: errMsgRefreshTokenRemovalPending}
	}
	s.store = store
	// Bind refresh enablement and its TTL to this passphrase-authenticated unlock.
	s.enableRefreshToken = req.EnableRefreshToken
	s.refreshTokenTTL = s.resolveRefreshTokenTTL(req.RefreshTokenTTL)
	s.logUnlocked(store, created)
	if s.enableRefreshToken {
		// Discard tokens unused past the TTL until the agent shuts down or is locked. The
		// sweep is bound to a cancelable child of the server context so LOCK can stop it
		// (see handleLock); otherwise a later unlock would start a second sweep while this
		// one keeps running.
		sweepCtx, cancel := context.WithCancel(ctx)
		s.sweepCancel = cancel
		s.startRefreshTokenSweep(sweepCtx, store, s.refreshTokenTTL)
	} else {
		// Refresh is off: drop any refresh tokens left by a previous refresh-enabled run.
		s.stripRefreshTokens(store)
	}
	return &agentapi.Response{OK: true, RefreshTokenEnabled: s.enableRefreshToken}
}

// logUnlocked logs the result of a successful unlock: the refresh-token state, and,
// when the unlock generated a new key, a warning about token files written under a
// previous key. Those files can't be decrypted with the new key (e.g. the key file was
// deleted while the tokens remained), so they are orphaned and will be re-minted.
// It is called with c.mu held.
func (s *Server) logUnlocked(store *tokenstore.Store, created bool) {
	if s.logger == nil {
		return
	}
	if created {
		s.logger.Info("generated a new agent key", "path", s.keyFile)
		if n := store.Len(); n > 0 {
			s.logger.Warn("found cached token files that predate the new agent key; they can't be decrypted and will be re-minted on the next get", "path", s.tokenDir, "count", n)
		}
	}
	s.logger.Info("agent unlocked", "refresh_token_enabled", s.enableRefreshToken)
}

// needsRefreshRemovalConfirmation reports whether this unlock would silently drop a
// still-valid stored refresh token: refresh is being turned off, the user has not yet
// confirmed the removal, and at least one stored token still carries a usable refresh
// token. When true, handleUnlock stays locked and asks the client to confirm.
func (s *Server) needsRefreshRemovalConfirmation(req *agentapi.Request, store *tokenstore.Store) bool {
	return !req.EnableRefreshToken && !req.ConfirmRefreshTokenRemoval && s.hasValidRefreshToken(store)
}

// hasValidRefreshToken reports whether any stored token carries a refresh token that is
// still valid (present and unexpired, per validRefreshToken). It gates the removal
// confirmation on unlock: an already-expired or absent refresh token is worthless, so
// dropping it needs no prompt. It is best-effort — a store or per-token read error is
// treated as "nothing valid found" so a glitch never forces a spurious prompt.
func (s *Server) hasValidRefreshToken(st *tokenstore.Store) bool {
	ids, err := st.ClientIDs()
	if err != nil {
		if s.logger != nil {
			slogerr.WithError(s.logger, err).Warn("list stored tokens to check for refresh tokens")
		}
		return false
	}
	for _, id := range ids {
		raw, ok, err := st.Get(id)
		if err != nil || !ok {
			continue
		}
		valid := s.validRefreshToken(raw) != ""
		scrub(raw)
		if valid {
			return true
		}
	}
	return false
}

// stripRefreshTokens removes the refresh token from every stored token, keeping the
// access token and its expiration. It is best-effort: per-token failures are logged and
// skipped rather than failing the unlock, since this is security cleanup, not a
// correctness requirement. The credentials are not revoked (that would send the user a
// notification email); they are simply dropped from the backend.
func (s *Server) stripRefreshTokens(st *tokenstore.Store) {
	ids, err := st.ClientIDs()
	if err != nil {
		if s.logger != nil {
			slogerr.WithError(s.logger, err).Warn("list stored tokens to strip refresh tokens")
		}
		return
	}
	for _, id := range ids {
		raw, ok, err := st.Get(id)
		if err != nil || !ok {
			// Unreadable/undecryptable/absent: nothing to strip.
			continue
		}
		stripped, changed := stripRefreshToken(raw)
		scrub(raw)
		if !changed {
			continue
		}
		if err := st.Set(id, stripped); err != nil {
			if s.logger != nil {
				slogerr.WithError(s.logger, err).Warn("rewrite a token without its refresh token", "client_id", id)
			}
		} else if s.logger != nil {
			s.logger.Info("dropped a stored refresh token because refresh is disabled", "client_id", id)
		}
		scrub(stripped)
	}
}

// stripRefreshToken returns raw with its refresh token and refresh-token expiration
// cleared, and whether anything changed. An unparsable token, or one that already has
// no refresh token, yields (nil, false).
func stripRefreshToken(raw json.RawMessage) (json.RawMessage, bool) {
	token := &storedToken{}
	if err := json.Unmarshal(raw, token); err != nil {
		return nil, false
	}
	if token.RefreshToken == "" && token.RefreshTokenExpirationDate.IsZero() {
		return nil, false
	}
	token.RefreshToken = ""
	token.RefreshTokenExpirationDate = time.Time{}
	//nolint:gosec // G117: re-serializing the stored token without its refresh token.
	b, err := json.Marshal(token)
	if err != nil {
		return nil, false
	}
	return b, true
}
