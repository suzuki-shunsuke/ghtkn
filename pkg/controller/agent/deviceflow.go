package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/go-github-device-flow/deviceflow"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// deviceFlowState holds the one-time code info for a device flow in progress so that
// any client polling GET can display it.
type deviceFlowState struct {
	userCode        string
	verificationURI string
	expiresIn       int
}

// deviceFlow returns the in-progress device flow state for clientID, if any.
func (c *Controller) deviceFlow(clientID string) (*deviceFlowState, bool) {
	c.statusMu.Lock()
	defer c.statusMu.Unlock()
	st, ok := c.status[clientID]
	return st, ok
}

// clearDeviceFlow removes the in-progress marker for clientID.
func (c *Controller) clearDeviceFlow(clientID string) {
	c.statusMu.Lock()
	defer c.statusMu.Unlock()
	delete(c.status, clientID)
}

// startDeviceFlow starts the server-side device flow for clientID. It requests a
// device code from GitHub, records the one-time code as in progress, and spawns a
// goroutine that polls for the access token and stores it. It returns the one-time
// code info the client displays.
//
// Any token already cached for clientID is deleted up front. The client starts a flow
// only after deciding the cached token is missing or stale, and the poll that waits for
// the result (AwaitDeviceFlow) returns the stored token without a freshness check, so
// the stale token must be gone: otherwise a failed flow would hand it back as if fresh.
func (c *Controller) startDeviceFlow(ctx context.Context, logger *slog.Logger, clientID string, enableRefreshToken bool) (*deviceFlowState, error) {
	// A flow that started while the caller was reading the store (or refreshing an
	// expiring token) is adopted here, before asking GitHub for a device code that would
	// then go unused. The claim below is still the authoritative check, since a flow can
	// also start during the request; this only keeps the common case off the network.
	if existing, ok := c.deviceFlow(clientID); ok {
		return existing, nil
	}

	//nolint:bodyclose // GetDeviceCode reads and closes the response body internally; it returns the decoded value.
	deviceCodeResp, _, _, err := c.client.GetDeviceCode(ctx, clientID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	state := &deviceFlowState{
		userCode:        deviceCodeResp.UserCode,
		verificationURI: deviceCodeResp.VerificationURI,
		expiresIn:       deviceCodeResp.ExpiresIn,
	}

	// Claim the in-progress slot atomically. If a start that raced with the request above
	// already claimed it, adopt that flow: return its (already displayed) one-time code
	// and start no second poller. Our own device code is simply left to expire unused.
	// This keeps a single poller per client ID even when two GETs race to start, so only
	// one flow stores the minted token and there are no competing pollers.
	c.statusMu.Lock()
	if existing, ok := c.status[clientID]; ok {
		c.statusMu.Unlock()
		return existing, nil
	}
	c.status[clientID] = state
	c.statusMu.Unlock()

	if st := c.tokenStore(); st != nil {
		// Best-effort: drop the stale token so a failed flow can't resurrect it. Only the
		// winner of the slot does this, so a concurrent adopter never deletes a token the
		// winner's poller may have just stored.
		if err := st.Delete(clientID); err != nil {
			slogerr.WithError(logger, err).Warn("delete the stale token before the device flow", "client_id", clientID)
		}
	}

	go func() {
		defer c.clearDeviceFlow(clientID)
		token, err := c.client.Poll(ctx, logger, clientID, deviceCodeResp, nil)
		if err != nil {
			slogerr.WithError(logger, err).Error("get access token via device flow", "client_id", clientID)
			return
		}
		st := c.tokenStore()
		if st == nil {
			logger.Error("backend is locked; discard the token minted by the device flow", "client_id", clientID)
			return
		}
		raw, err := c.encodeToken(token, enableRefreshToken)
		if err != nil {
			slogerr.WithError(logger, err).Error("encode the minted access token", "client_id", clientID)
			return
		}
		defer scrub(raw)
		// Store the token before clearing the in-progress marker (deferred) so a poll
		// never sees "not in progress" before the token is readable.
		if err := st.Set(clientID, raw); err != nil {
			slogerr.WithError(logger, err).Error("set access token to backend", "client_id", clientID)
		}
	}()

	return state, nil
}

// encodeToken converts a device flow access token into the JSON the agent stores: the
// token value plus an absolute expiration date computed from ExpiresIn. When
// enableRefreshToken is set it also stores the refresh token and its own (longer)
// expiration computed from RefreshTokenExpiresIn.
func (c *Controller) encodeToken(token *deviceflow.AccessToken, enableRefreshToken bool) (json.RawMessage, error) {
	now := time.Now()
	ac := &storedToken{
		AccessToken:    token.AccessToken,
		ExpirationDate: now.Add(time.Duration(token.ExpiresIn) * time.Second),
	}
	if enableRefreshToken {
		ac.RefreshToken = token.RefreshToken
		ac.RefreshTokenExpirationDate = now.Add(time.Duration(token.RefreshTokenExpiresIn) * time.Second)
	}

	//nolint:gosec // G117: the access token is intentionally serialized so tokenstore can persist it encrypted at rest.
	raw, err := json.Marshal(ac)
	if err != nil {
		return nil, fmt.Errorf("marshal the access token as JSON: %w", err)
	}
	return raw, nil
}
