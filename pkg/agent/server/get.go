package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/tokenstore"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// maxTokenLifetime is the longest a GitHub App user access token can be valid: GitHub
// issues them for at most 8 hours. A minExpiration greater than this cannot be satisfied
// by any real token, so it means "regenerate regardless" (as 'ghtkn auth' relies on).
const maxTokenLifetime = 8 * time.Hour

// tokenValid reports whether the stored token is still valid for at least
// minExpiration from now. An unparsable token is treated as invalid so it is
// re-minted rather than served.
func (s *Server) tokenValid(raw json.RawMessage, minExpiration time.Duration) bool {
	// Parse only the expiration; reading it must not materialize the access/refresh
	// token as a Go string.
	token := &struct {
		ExpirationDate time.Time `json:"expiration_date"`
	}{}
	if err := json.Unmarshal(raw, token); err != nil {
		return false
	}
	if token.ExpirationDate.IsZero() {
		// The token never expires (a GitHub App with user-token expiration disabled).
		// It is valid on its own, but a minExpiration beyond the maximum token lifetime
		// is a request to regenerate regardless (ghtkn auth forcing a fresh token so a
		// revoked one is replaced), so it counts as invalid then.
		return minExpiration <= maxTokenLifetime
	}
	// Valid when it does not expire within minExpiration, i.e. now+minExpiration is
	// not after the expiration date.
	return !time.Now().Add(minExpiration).After(token.ExpirationDate)
}

// pendingResponse builds the "device flow in progress" response, echoing the one-time
// code so the client can display it.
func pendingResponse(state *deviceFlowState) *agentapi.Response {
	return &agentapi.Response{
		OK:              true,
		Pending:         true,
		UserCode:        state.userCode,
		VerificationURI: state.verificationURI,
		ExpiresIn:       state.expiresIn,
	}
}

// deviceFlowResult answers a poll that waits for a device-flow result: it returns the
// stored token as is, or RespNotFound when the flow ended without storing one. This is
// only reached once the flow's in-progress marker is gone, and a failed flow leaves the
// marker with an errMsg that handleGet turns into an error before here (the marker is
// cleared only on a clean completion, which stores the freshly minted token over any
// previous one). So a token present here is that freshly minted one, even though the
// pre-flow token was never deleted up front.
func (s *Server) deviceFlowResult(st *tokenstore.Store, clientID string) *agentapi.Response {
	token, ok, resp := s.readStoredToken(st, clientID)
	defer scrub(token)
	switch {
	case resp != nil:
		return resp
	case ok:
		return tokenResponse(token)
	default:
		return &agentapi.Response{Error: agentapi.RespNotFound}
	}
}

// scrub best-effort zeroes a decrypted token buffer once it is no longer needed, to
// shorten how long a plaintext access/refresh token (which carries a scannable
// "ghu_"/"ghr_" prefix) lives in memory. It is not a guarantee: the JSON round-trip in
// tokenResponse creates string copies the runtime may retain until GC. scrub(nil) is a
// no-op.
func scrub(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// withWarning attaches a non-fatal, security-relevant warning to a response the client
// must surface to the user (see refreshAccessToken). An empty warning leaves resp as is.
func withWarning(resp *agentapi.Response, warning string) *agentapi.Response {
	if warning != "" {
		resp.Warning = warning
	}
	return resp
}

// incidentWarning is the message shown to the user when a refresh token that is still
// within its expiration fails to refresh: a strong signal that the refresh token may
// have leaked and been used elsewhere, or that the app's authorization was revoked.
func incidentWarning(clientID string) string {
	return fmt.Sprintf("a still-valid refresh token failed to refresh for GitHub App %s. "+
		"This can happen if the refresh token was leaked and used from elsewhere, or the app's "+
		"authorization was revoked. If you did not expect this, treat it as a possible security "+
		"incident: revoke this app's tokens, or suspend or uninstall the GitHub App.", clientID)
}

// tokenResponse builds a successful GET response that carries only the client-facing
// fields of a stored token. The refresh token and its expiration are stripped so they
// never leave the agent: they stay in the encrypted store and are used only for
// server-side refresh.
func tokenResponse(raw json.RawMessage) *agentapi.Response {
	// Keep the access token and expiration as raw JSON bytes so the access token is
	// never materialized as a Go string here; only the two client-facing fields are
	// copied, so the refresh token stays server-side.
	token := &struct {
		AccessToken    json.RawMessage `json:"access_token"`
		ExpirationDate json.RawMessage `json:"expiration_date"`
	}{}
	if err := json.Unmarshal(raw, token); err != nil {
		return &agentapi.Response{Error: fmt.Sprintf("%s: %s", errMsgGet, err)}
	}
	// The extracted access-token bytes are copied into the marshaled response below;
	// zero this intermediate copy once done.
	defer scrub(token.AccessToken)
	// Only the access token is serialized here; the refresh token is kept server-side.
	b, err := json.Marshal(token)
	if err != nil {
		return &agentapi.Response{Error: fmt.Sprintf("%s: %s", errMsgGet, err)}
	}
	return &agentapi.Response{OK: true, Token: b}
}

// handleGet returns the cached token for the request's client ID, or drives the
// server-side device flow when the client asked for it.
//
// The order: an in-progress flow is reported first (so a poll never returns a stale
// token while the flow runs), then a cached token that is still valid for the
// requested MinExpiration is returned, and only when there is no valid token does an
// explicit start request begin the flow (otherwise a not-found miss). Checking the
// stored token before starting a flow means a token minted concurrently by another
// client is returned instead of starting a redundant flow.
func (s *Server) handleGet(ctx context.Context, req *agentapi.Request, enableRefreshToken bool) *agentapi.Response {
	st := s.tokenStore()
	if st == nil {
		return &agentapi.Response{Error: agentapi.RespLocked}
	}

	// A device flow is recorded for this client ID.
	if state, ok := s.deviceFlow(req.ClientID); ok {
		switch {
		case state.errMsg == "":
			// Still running: report progress and echo the one-time code so a client that
			// just arrived can display it too.
			return pendingResponse(state)
		case req.AwaitDeviceFlow:
			// The flow this client is waiting on failed. Tell it so, rather than letting
			// it fall through to the pre-flow token that was never deleted.
			return &agentapi.Response{Error: state.errMsg}
		default:
			// A non-waiting request: the recorded failure is stale for it, so drop it and
			// continue. A StartDeviceFlow request starts a fresh flow below.
			s.clearDeviceFlow(req.ClientID)
		}
	}

	// Polling for the result of a flow this client started. A failed flow was already
	// turned into an error above (its marker carries an errMsg), so reaching here means
	// the flow completed and stored its token, overwriting any pre-flow one. Return that
	// token as is (no freshness check); its absence means the flow ended without a token.
	if req.AwaitDeviceFlow {
		return s.deviceFlowResult(st, req.ClientID)
	}

	// Serve a valid cached token, or silently refresh an expiring one. A nil result
	// means there is no usable token and the device flow must run. warning carries a
	// security-relevant message (e.g. a valid refresh token that failed to refresh) that
	// must reach the user regardless of which outcome is ultimately returned.
	resp, warning := s.cachedToken(ctx, st, req, enableRefreshToken)
	if resp != nil {
		return withWarning(resp, warning)
	}

	// No valid cached token. Start the flow only when the client asked to; the server
	// mints and stores the token and the client polls (AwaitDeviceFlow) until ready.
	if req.StartDeviceFlow {
		state, err := s.startDeviceFlow(ctx, s.logger, req.ClientID, enableRefreshToken)
		if err != nil {
			return withWarning(&agentapi.Response{Error: fmt.Sprintf("%s: %s", errMsgStartDeviceFlow, err)}, warning)
		}
		return withWarning(pendingResponse(state), warning)
	}
	return withWarning(&agentapi.Response{Error: agentapi.RespNotFound}, warning)
}

// cachedToken serves the stored token for the request: the token itself when it is still
// valid for MinExpiration, or a silently refreshed token when it is expiring, refresh is
// enabled, and a valid refresh token is stored. It returns a nil response to fall through
// to the device flow (no token, expired with no usable refresh), or an error response on
// a store error. The second result is a security warning to surface to the user (see
// refreshAccessToken); it may be non-empty even when the response is nil.
func (s *Server) cachedToken(ctx context.Context, st *tokenstore.Store, req *agentapi.Request, enableRefreshToken bool) (*agentapi.Response, string) {
	token, ok, resp := s.readStoredToken(st, req.ClientID)
	if resp != nil {
		return resp, ""
	}
	if !ok {
		return nil, ""
	}
	// The decrypted token only needs to live for this call; scrub it before returning.
	defer scrub(token)
	if s.tokenValid(token, req.MinExpiration) {
		return tokenResponse(token), ""
	}
	if enableRefreshToken {
		return s.refreshAccessToken(ctx, st, req.ClientID, req.MinExpiration)
	}
	return nil, ""
}

// lockRefresh acquires the per-client refresh lock for clientID, waiting while another
// GET for the same client is refreshing. It returns the release function, or ctx's error
// if ctx is done first (the agent is shutting down). A channel is used instead of a
// sync.Mutex so the wait can be abandoned on ctx cancellation: the holder may be blocked
// on a GitHub request for up to githubHTTPTimeout.
func (s *Server) lockRefresh(ctx context.Context, clientID string) (func(), error) {
	s.refreshLocksMu.Lock()
	sem, ok := s.refreshLocks[clientID]
	if !ok {
		sem = make(chan struct{}, 1)
		s.refreshLocks[clientID] = sem
	}
	s.refreshLocksMu.Unlock()
	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err() //nolint:wrapcheck
	}
}

// reloadExpiringToken re-reads the stored token under the refresh lock. It returns the
// token when it still needs a refresh (not valid for minExpiration); the caller scrubs it.
// Otherwise it returns nil and the response to serve: the token a GET that held the lock
// before this one already refreshed, an error response on a store error, or nil (no token,
// fall back to the device flow).
func (s *Server) reloadExpiringToken(st *tokenstore.Store, clientID string, minExpiration time.Duration) (json.RawMessage, *agentapi.Response) {
	raw, ok, resp := s.readStoredToken(st, clientID)
	if resp != nil || !ok {
		return nil, resp
	}
	if s.tokenValid(raw, minExpiration) {
		defer scrub(raw)
		return nil, tokenResponse(raw)
	}
	return raw, nil
}

// validRefreshToken returns the stored refresh token if it is present and still valid
// (per the clock), or "" otherwise.
func (s *Server) validRefreshToken(raw json.RawMessage) string {
	// Parse only the refresh token and its expiration; the access token is not read
	// here, so it is never materialized as a Go string.
	token := &struct {
		RefreshToken               string    `json:"refresh_token"`
		RefreshTokenExpirationDate time.Time `json:"refresh_token_expiration_date"`
	}{}
	if err := json.Unmarshal(raw, token); err != nil {
		return ""
	}
	if token.RefreshToken == "" || token.RefreshTokenExpirationDate.IsZero() {
		return ""
	}
	if !time.Now().Before(token.RefreshTokenExpirationDate) {
		return "" // the refresh token has expired
	}
	return token.RefreshToken
}

// refreshAccessToken tries to silently refresh an expiring access token using a stored,
// still-valid refresh token. It returns a response carrying the new token on success, or
// a nil response to fall back to the device flow (no usable refresh token, or the refresh
// request failed). On success the refreshed token (which GitHub returns with a rotated
// refresh token) is stored before it is returned.
//
// The second result is a warning to surface to the user. When the refresh token is still
// within its expiration yet the refresh fails, that is a possible incident (the refresh
// token may have leaked or been revoked), so the warning is set even though the response
// falls back to the device flow.
//
// A rotating refresh token is single-use, so refreshes are serialized per client ID (see
// lockRefresh). Under the lock the stored token is re-read and re-checked against
// minExpiration: when concurrent GETs find the same expiring token, the first refreshes
// it and the rest are served the token it stored, instead of each spending the already
// rotated refresh token and raising a false incident warning.
func (s *Server) refreshAccessToken(ctx context.Context, st *tokenstore.Store, clientID string, minExpiration time.Duration) (*agentapi.Response, string) {
	unlock, err := s.lockRefresh(ctx, clientID)
	if err != nil {
		return &agentapi.Response{Error: fmt.Sprintf("%s: %s", errMsgGet, err)}, ""
	}
	defer unlock()

	raw, resp := s.reloadExpiringToken(st, clientID, minExpiration)
	if raw == nil {
		return resp, ""
	}
	defer scrub(raw)

	refreshToken := s.validRefreshToken(raw)
	if refreshToken == "" {
		return nil, ""
	}

	//nolint:bodyclose // RefreshToken reads and closes the response body internally; it returns the decoded value.
	newToken, _, _, err := s.client.RefreshToken(ctx, clientID, refreshToken)
	if err != nil {
		// The refresh token was still valid, and no other GET in this agent can have spent it
		// (refreshes are serialized per client), yet the refresh failed: warn the user of a
		// possible leak/revocation, then fall back to the device flow.
		if s.logger != nil {
			slogerr.WithError(s.logger, err).Error("a still-valid refresh token failed to refresh; possible incident", "client_id", clientID)
		}
		return nil, incidentWarning(clientID)
	}

	fresh, err := s.encodeToken(newToken, true)
	if err != nil {
		if s.logger != nil {
			slogerr.WithError(s.logger, err).Error("encode the refreshed access token", "client_id", clientID)
		}
		return nil, ""
	}
	defer scrub(fresh)
	// Return the refreshed token so this get succeeds even if the store write fails below.
	if err := st.Set(clientID, fresh); err != nil {
		if s.logger != nil {
			slogerr.WithError(s.logger, err).Warn("store the refreshed access token", "client_id", clientID)
		}
		// GitHub rotated the refresh token, so the copy in `fresh` (returned but not
		// persisted) is the only live one and the stored token's refresh token is now spent.
		// Left in place, the next get would try that dead refresh token, fail, and raise a
		// false incident warning. Drop the stale stored token so the next get re-authenticates
		// via the device flow instead.
		s.dropStaleAfterFailedStore(st, clientID, minExpiration)
	}
	return tokenResponse(fresh), ""
}

// dropStaleAfterFailedStore best-effort discards the cached token for clientID after a
// refresh whose store write failed. Because GitHub rotates the refresh token, a failed
// store leaves the stored token carrying a now-spent refresh token; discarding it makes
// the next get fall back to the device flow instead of trying the dead refresh token and
// raising a false incident warning. The delete is conditional (DeleteIf): it removes the
// token only while it is still not valid for minExpiration, so a fresher token a concurrent
// refresh or device flow may have stored in the meantime is left in place.
//
// This helps when the delete can still succeed even though the store write could not, e.g.
// a full disk, where os.Remove frees the directory entry without needing space. When the
// failure is directory-level (permissions) the delete also fails and the stale token
// remains, but then the agent cannot store any token at all, a louder problem than one
// false incident warning, and this is no worse than leaving it in place.
func (s *Server) dropStaleAfterFailedStore(st *tokenstore.Store, clientID string, minExpiration time.Duration) {
	if _, err := st.DeleteIf(clientID, func(raw json.RawMessage) bool {
		return !s.tokenValid(raw, minExpiration)
	}); err != nil && s.logger != nil {
		slogerr.WithError(s.logger, err).Warn("drop the stale cached token after a failed refresh store", "client_id", clientID)
	}
}
