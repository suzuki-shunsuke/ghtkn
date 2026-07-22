package agent

import (
	"context"
	"encoding/json"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// handleRevoke revokes the tokens cached for the request's client IDs in one batch
// and deletes them. Client IDs with no stored (or unreadable) token are skipped. The
// response reports which client IDs could not be revoked (their credential may be
// live) and which were revoked but could not be deleted (a cleanup issue), so the
// client can classify each. When the batch revocation call fails, no token is deleted
// and every attempted client ID is reported as a revoke failure.
func (c *Controller) handleRevoke(ctx context.Context, req *agentapi.Request) *agentapi.Response {
	st := c.tokenStore()
	if st == nil {
		return &agentapi.Response{Error: agentapi.RespLocked}
	}

	tokens, attempted, revokeFailed := c.collectRevocableTokens(st, req.ClientIDs)
	if len(tokens) == 0 {
		return &agentapi.Response{OK: true, RevokeFailed: revokeFailed}
	}
	if err := c.revoker.Revoke(ctx, tokens); err != nil {
		// The batch call failed, so none of the credentials were revoked and none are
		// deleted: report every attempted client ID as a revoke failure.
		if c.logger != nil {
			slogerr.WithError(c.logger, err).Warn("revoke stored tokens")
		}
		return &agentapi.Response{OK: true, RevokeFailed: append(revokeFailed, attempted...)}
	}
	return &agentapi.Response{OK: true, RevokeFailed: revokeFailed, CleanupFailed: c.deleteRevoked(st, attempted)}
}

// collectRevocableTokens reads the stored token for each client ID. It returns the
// access tokens to revoke, the client IDs they belong to (same order), and the client
// IDs whose token could not be read (which the caller reports as revoke failures).
// Client IDs with no stored token are skipped.
func (c *Controller) collectRevocableTokens(st *tokenstore.Store, clientIDs []string) (tokens, attempted, revokeFailed []string) {
	for _, clientID := range clientIDs {
		raw, ok, resp := c.readStoredToken(st, clientID)
		if resp != nil {
			// The token couldn't be read, so it can't be revoked; it may be live.
			revokeFailed = append(revokeFailed, clientID)
			continue
		}
		if !ok {
			continue // nothing stored for this client: nothing to revoke.
		}
		token := &storedToken{}
		err := json.Unmarshal(raw, token)
		// The decrypted plaintext (access + refresh token) is no longer needed once
		// parsed; the extracted string fields below are what get revoked.
		scrub(raw)
		if err != nil {
			revokeFailed = append(revokeFailed, clientID)
			continue
		}
		tokens = append(tokens, token.AccessToken)
		if token.RefreshToken != "" {
			tokens = append(tokens, token.RefreshToken)
		}
		attempted = append(attempted, clientID)
	}
	return tokens, attempted, revokeFailed
}

// deleteRevoked deletes the stored copies of already-revoked tokens and returns the
// client IDs whose deletion failed (a cleanup issue, not a revoke failure).
func (c *Controller) deleteRevoked(st *tokenstore.Store, clientIDs []string) []string {
	var cleanupFailed []string
	for _, clientID := range clientIDs {
		if err := st.Delete(clientID); err != nil {
			if c.logger != nil {
				slogerr.WithError(c.logger, err).Warn("delete a revoked token", "client_id", clientID)
			}
			cleanupFailed = append(cleanupFailed, clientID)
		}
	}
	return cleanupFailed
}
