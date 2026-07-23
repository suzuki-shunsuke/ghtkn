package agent

import (
	"context"
	"encoding/json"
	"errors"
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
	// errMsg is empty while the flow runs. It is set when the flow ends without storing
	// a token (the one-time code expired, the poll failed, or the token could not be
	// stored), so a poll waiting on this flow (AwaitDeviceFlow) is told it failed instead
	// of being handed the pre-flow token, which is no longer deleted up front.
	errMsg string
}

// deviceFlow returns a snapshot of the device flow state for clientID, if any. It copies
// the state under the lock so a caller can read errMsg without racing the flow goroutine
// that sets it.
func (c *Controller) deviceFlow(clientID string) (*deviceFlowState, bool) {
	c.statusMu.Lock()
	defer c.statusMu.Unlock()
	st, ok := c.status[clientID]
	if !ok {
		return nil, false
	}
	snapshot := *st
	return &snapshot, true
}

// clearDeviceFlow removes the in-progress marker for clientID.
func (c *Controller) clearDeviceFlow(clientID string) {
	c.statusMu.Lock()
	defer c.statusMu.Unlock()
	delete(c.status, clientID)
}

// failDeviceFlow marks the recorded flow for clientID as failed, so a poll waiting on it
// (AwaitDeviceFlow) returns an error rather than the pre-flow token. It is a no-op when
// no flow is recorded (already cleared).
func (c *Controller) failDeviceFlow(clientID string) {
	c.statusMu.Lock()
	defer c.statusMu.Unlock()
	if st, ok := c.status[clientID]; ok {
		st.errMsg = errMsgDeviceFlowFailed
	}
}

// startDeviceFlow starts the server-side device flow for clientID. It requests a
// device code from GitHub, records the one-time code as in progress, and spawns a
// goroutine that polls for the access token and stores it. It returns the one-time
// code info the client displays.
//
// The pre-flow token is left in place. A poll that waits for the result (AwaitDeviceFlow)
// returns the stored token without a freshness check, so a failed flow must not leave the
// old token to be handed back as if fresh: instead the goroutine marks the flow failed
// (see failDeviceFlow) on any exit that does not store a new token, and handleGet turns
// that into an error for the waiting client. Not deleting the token means a concurrent
// client, or a later one after an abandoned flow, keeps using it.
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

	go func() {
		// Any exit from pollAndStore without a stored token returns an error, so the flow
		// is marked failed in one place: a new step that can fail just returns its error
		// and cannot forget to mark it. Only a clean completion clears the marker.
		if err := c.pollAndStore(ctx, logger, clientID, deviceCodeResp, enableRefreshToken); err != nil {
			slogerr.WithError(logger, err).Error("the device flow did not store a token", "client_id", clientID)
			c.failDeviceFlow(clientID)
			return
		}
		c.clearDeviceFlow(clientID)
	}()

	return state, nil
}

// pollAndStore polls the device flow to completion and stores the minted token. It
// returns an error on any exit that does not store a token, so startDeviceFlow can mark
// the flow failed without a per-path check.
func (c *Controller) pollAndStore(ctx context.Context, logger *slog.Logger, clientID string, deviceCode *deviceflow.DeviceCodeResponse, enableRefreshToken bool) error {
	token, err := c.client.Poll(ctx, logger, clientID, deviceCode, nil)
	if err != nil {
		return fmt.Errorf("get an access token via the device flow: %w", err)
	}
	st := c.tokenStore()
	if st == nil {
		return errors.New("the backend is locked, so the minted token was discarded")
	}
	raw, err := c.encodeToken(token, enableRefreshToken)
	if err != nil {
		return fmt.Errorf("encode the minted access token: %w", err)
	}
	defer scrub(raw)
	// Store the token before returning (the caller clears the in-progress marker only
	// after this), so a poll never sees "not in progress" before the token is readable.
	if err := st.Set(clientID, raw); err != nil {
		return fmt.Errorf("store the minted access token: %w", err)
	}
	return nil
}

// encodeToken converts a device flow access token into the JSON the agent stores: the
// token value plus an absolute expiration date computed from ExpiresIn. When
// enableRefreshToken is set it also stores the refresh token and its own (longer)
// expiration computed from RefreshTokenExpiresIn.
func (c *Controller) encodeToken(token *deviceflow.AccessToken, enableRefreshToken bool) (json.RawMessage, error) {
	now := time.Now()
	ac := &storedToken{
		AccessToken: token.AccessToken,
	}
	// expires_in=0 means the token never expires (a GitHub App with user-token
	// expiration disabled); leave ExpirationDate the zero time, which the expiry checks
	// read as never-expiring rather than already-expired.
	if token.ExpiresIn != 0 {
		ac.ExpirationDate = now.Add(time.Duration(token.ExpiresIn) * time.Second)
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
