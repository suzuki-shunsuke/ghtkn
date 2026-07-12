package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// tokenValid reports whether the stored token is still valid for at least
// minExpiration from now. An unparsable token is treated as invalid so it is
// re-minted rather than served.
func (c *Controller) tokenValid(raw json.RawMessage, minExpiration time.Duration) bool {
	token := &pubapi.AccessToken{}
	if err := json.Unmarshal(raw, token); err != nil {
		return false
	}
	// Valid when it does not expire within minExpiration, i.e. now+minExpiration is
	// not after the expiration date.
	return !c.now().Add(minExpiration).After(token.ExpirationDate)
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
// stored token as is (the flow deleted any stale token first, so a present token is the
// freshly minted one), or RespNotFound when the flow ended without storing one.
func (c *Controller) deviceFlowResult(st *tokenstore.Store, clientID string) *agentapi.Response {
	token, ok, resp := c.readStoredToken(st, clientID)
	switch {
	case resp != nil:
		return resp
	case ok:
		return tokenResponse(token)
	default:
		return &agentapi.Response{Error: agentapi.RespNotFound}
	}
}

// tokenResponse builds a successful GET response that carries only the client-facing
// fields of a stored token. The refresh token and its expiration are stripped so they
// never leave the agent: they stay in the encrypted store and are used only for
// server-side refresh.
func tokenResponse(raw json.RawMessage) *agentapi.Response {
	token := &struct {
		AccessToken    string    `json:"access_token"`
		ExpirationDate time.Time `json:"expiration_date"`
	}{}
	if err := json.Unmarshal(raw, token); err != nil {
		return &agentapi.Response{Error: fmt.Sprintf("%s: %s", errMsgGet, err)}
	}
	//nolint:gosec // G117: serializing only the access token; the refresh token is kept server-side.
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
func (c *Controller) handleGet(ctx context.Context, req *agentapi.Request, enableRefreshToken bool) *agentapi.Response {
	st := c.tokenStore()
	if st == nil {
		return &agentapi.Response{Error: agentapi.RespLocked}
	}

	// A device flow is already running for this client ID. Report progress and echo
	// the one-time code so a client that just arrived can display it too.
	if state, ok := c.deviceFlow(req.ClientID); ok {
		return pendingResponse(state)
	}

	// Polling for the result of a flow this client started: the flow deleted any stale
	// token up front, so a stored token here is the freshly minted one. Return it as is
	// (no freshness check); its absence means the flow ended without a token.
	if req.AwaitDeviceFlow {
		return c.deviceFlowResult(st, req.ClientID)
	}

	// Serve a valid cached token, or silently refresh an expiring one. A nil result
	// means there is no usable token and the device flow must run.
	if resp := c.cachedToken(ctx, st, req, enableRefreshToken); resp != nil {
		return resp
	}

	// No valid cached token. Start the flow only when the client asked to; the server
	// mints and stores the token and the client polls (AwaitDeviceFlow) until ready.
	if req.StartDeviceFlow {
		state, err := c.startDeviceFlow(ctx, c.logger, req.ClientID, enableRefreshToken)
		if err != nil {
			return &agentapi.Response{Error: fmt.Sprintf("%s: %s", errMsgStartDeviceFlow, err)}
		}
		return pendingResponse(state)
	}
	return &agentapi.Response{Error: agentapi.RespNotFound}
}

// cachedToken serves the stored token for the request: the token itself when it is still
// valid for MinExpiration, or a silently refreshed token when it is expiring, refresh is
// enabled, and a valid refresh token is stored. It returns nil to fall through to the
// device flow (no token, expired with no usable refresh), or an error response on a
// store error.
func (c *Controller) cachedToken(ctx context.Context, st *tokenstore.Store, req *agentapi.Request, enableRefreshToken bool) *agentapi.Response {
	token, ok, resp := c.readStoredToken(st, req.ClientID)
	if resp != nil {
		return resp
	}
	if !ok {
		return nil
	}
	if c.tokenValid(token, req.MinExpiration) {
		return tokenResponse(token)
	}
	if enableRefreshToken {
		return c.refreshAccessToken(ctx, st, req.ClientID, token)
	}
	return nil
}

// validRefreshToken returns the stored refresh token if it is present and still valid
// (per the controller clock), or "" otherwise.
func (c *Controller) validRefreshToken(raw json.RawMessage) string {
	token := &pubapi.AccessToken{}
	if err := json.Unmarshal(raw, token); err != nil {
		return ""
	}
	if token.RefreshToken == "" || token.RefreshTokenExpirationDate.IsZero() {
		return ""
	}
	if !c.now().Before(token.RefreshTokenExpirationDate) {
		return "" // the refresh token has expired
	}
	return token.RefreshToken
}

// refreshAccessToken tries to silently refresh an expiring access token using a stored,
// still-valid refresh token. It returns a response carrying the new token on success, or
// nil to fall back to the device flow (no usable refresh token, or the refresh request
// failed). On success the refreshed token (which GitHub returns with a rotated refresh
// token) is stored before it is returned.
func (c *Controller) refreshAccessToken(ctx context.Context, st *tokenstore.Store, clientID string, raw json.RawMessage) *agentapi.Response {
	refreshToken := c.validRefreshToken(raw)
	if refreshToken == "" {
		return nil
	}

	//nolint:bodyclose // RefreshToken reads and closes the response body internally; it returns the decoded value.
	newToken, _, _, err := c.client.RefreshToken(ctx, clientID, refreshToken)
	if err != nil {
		if c.logger != nil {
			slogerr.WithError(c.logger, err).Warn("refresh the access token; falling back to the device flow", "client_id", clientID)
		}
		return nil
	}

	fresh, err := c.encodeToken(newToken, true)
	if err != nil {
		if c.logger != nil {
			slogerr.WithError(c.logger, err).Error("encode the refreshed access token", "client_id", clientID)
		}
		return nil
	}
	// A store failure is not fatal: return the refreshed token so the client gets a
	// working one this run (the next get simply refreshes again).
	if err := st.Set(clientID, fresh); err != nil {
		if c.logger != nil {
			slogerr.WithError(c.logger, err).Warn("store the refreshed access token", "client_id", clientID)
		}
	}
	return tokenResponse(fresh)
}
