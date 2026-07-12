package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/go-github-device-flow/deviceflow"
)

// refreshRoundTripper is a fake HTTP transport for the token endpoint. It records
// whether it was called so tests can assert a refresh was (not) attempted.
type refreshRoundTripper struct {
	called bool
	status int
	body   string
	err    error
}

func (f *refreshRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.status, Body: io.NopCloser(strings.NewReader(f.body)), Header: make(http.Header)}, nil
}

// setClientTransport points the controller's device-flow client at rt.
func setClientTransport(c *Controller, rt http.RoundTripper) {
	c.client = deviceflow.New(&deviceflow.Input{HTTPClient: &http.Client{Transport: rt}})
}

// TestController_handleGet_pendingFlow verifies that a GET for a client with a device
// flow already in progress reports progress and echoes the one-time code, without
// touching the network.
func TestController_handleGet_pendingFlow(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	const clientID = "Iv1.deviceflow"
	c.status[clientID] = &deviceFlowState{
		userCode:        "ABCD-1234",
		verificationURI: "https://github.com/login/device",
		expiresIn:       900,
	}
	got := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, false)
	want := &agentapi.Response{
		OK:              true,
		Pending:         true,
		UserCode:        "ABCD-1234",
		VerificationURI: "https://github.com/login/device",
		ExpiresIn:       900,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("in-progress flow (-want +got):\n%s", diff)
	}
}

// TestController_handleGet_missNoFlow verifies that a plain GET (StartDeviceFlow false)
// for a client with no token and no in-progress flow is a not-found miss and does not
// start a device flow.
func TestController_handleGet_missNoFlow(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	got := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: "Iv1.notoken"}, false)
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
		t.Fatalf("plain GET miss (-want +got):\n%s", diff)
	}
	if len(c.status) != 0 {
		t.Fatalf("a plain GET must not start a flow; status has %d entries, want 0", len(c.status))
	}
}

// TestController_handleGet_expired verifies that a plain GET for a token whose
// expiration date is in the past (relative to a fixed now) is a not-found miss.
func TestController_handleGet_expired(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	c.now = func() time.Time { return time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC) }
	const clientID = "Iv1.expired"
	if err := c.store.Set(clientID, json.RawMessage(`{"access_token":"abc","expiration_date":"2020-01-01T00:00:00Z"}`)); err != nil {
		t.Fatal(err)
	}
	got := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, false)
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
		t.Fatalf("expired GET (-want +got):\n%s", diff)
	}
}

// TestController_handleGet_minExpiration verifies that MinExpiration is honored: a token
// expiring 30m after now is returned for a 10m requirement but not for a 1h requirement.
func TestController_handleGet_minExpiration(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	const clientID = "Iv1.minexp"
	seeded := fmt.Sprintf(`{"access_token":"abc","expiration_date":"%s"}`, now.Add(30*time.Minute).Format(time.RFC3339))
	if err := c.store.Set(clientID, json.RawMessage(seeded)); err != nil {
		t.Fatal(err)
	}

	within := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID, MinExpiration: 10 * time.Minute}, false)
	if diff := cmp.Diff(&agentapi.Response{OK: true, Token: []byte(seeded)}, within); diff != "" {
		t.Fatalf("GET with 10m MinExpiration (-want +got):\n%s", diff)
	}

	beyond := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID, MinExpiration: time.Hour}, false)
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, beyond); diff != "" {
		t.Fatalf("GET with 1h MinExpiration (-want +got):\n%s", diff)
	}
}

// TestController_handleGet_awaitReturnsUnfreshToken verifies that a poll waiting for a
// device-flow result (AwaitDeviceFlow) returns the stored token as is, without the
// freshness check, so a freshly minted but short-lived token (e.g. a non-expiring
// GitHub App token, stored with expiration_date = mint time) is still handed back.
func TestController_handleGet_awaitReturnsUnfreshToken(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	c.now = func() time.Time { return time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC) }
	const clientID = "Iv1.await"
	// expiration_date is in the past relative to c.now, so a plain GET would treat it
	// as a miss; the await poll must still return it.
	seeded := `{"access_token":"minted","expiration_date":"2020-01-01T00:00:00Z"}`
	if err := c.store.Set(clientID, json.RawMessage(seeded)); err != nil {
		t.Fatal(err)
	}
	got := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID, AwaitDeviceFlow: true}, false)
	if diff := cmp.Diff(&agentapi.Response{OK: true, Token: []byte(seeded)}, got); diff != "" {
		t.Fatalf("await GET (-want +got):\n%s", diff)
	}
}

// TestController_handleGet_awaitNoToken verifies that a poll finding no stored token
// (the flow ended without minting one) reports not found.
func TestController_handleGet_awaitNoToken(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	got := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: "Iv1.none", AwaitDeviceFlow: true}, false)
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
		t.Fatalf("await GET without a token (-want +got):\n%s", diff)
	}
}

// seedExpiredWithRefresh stores an expired access token whose refresh token is valid
// until refreshValidUntil, and returns the seeded raw JSON.
func seedExpiredWithRefresh(t *testing.T, c *Controller, clientID string, refreshValidUntil time.Time) {
	t.Helper()
	seeded := fmt.Sprintf(`{"access_token":"old","expiration_date":"2020-01-01T00:00:00Z","refresh_token":"old-refresh","refresh_token_expiration_date":"%s"}`, refreshValidUntil.Format(time.RFC3339))
	if err := c.store.Set(clientID, json.RawMessage(seeded)); err != nil {
		t.Fatal(err)
	}
}

// TestController_handleGet_refresh verifies that an expired access token with a valid
// refresh token is silently refreshed (no device flow), and the rotated token is stored.
func TestController_handleGet_refresh(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	rt := &refreshRoundTripper{
		status: http.StatusOK,
		body:   `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":28800,"refresh_token_expires_in":15897600}`,
	}
	setClientTransport(c, rt)
	const clientID = "Iv1.refresh"
	seedExpiredWithRefresh(t, c, clientID, now.Add(24*time.Hour))

	got := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, true)

	// The response carries only the access token; the refresh token stays server-side.
	//nolint:gosec // G117: serializing a token in a test to build the expected bytes.
	wantResp, err := json.Marshal(&struct {
		AccessToken    string    `json:"access_token"`
		ExpirationDate time.Time `json:"expiration_date"`
	}{AccessToken: "new-access", ExpirationDate: now.Add(28800 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(&agentapi.Response{OK: true, Token: wantResp}, got); diff != "" {
		t.Fatalf("refreshed GET (-want +got):\n%s", diff)
	}
	if !rt.called {
		t.Fatal("the refresh endpoint was not called")
	}
	// The rotated token, including the refresh token, is persisted in the store.
	//nolint:gosec // G117: serializing a token in a test to build the expected stored bytes.
	wantStored, err := json.Marshal(&pubapi.AccessToken{
		AccessToken:                "new-access",
		ExpirationDate:             now.Add(28800 * time.Second),
		RefreshToken:               "new-refresh",
		RefreshTokenExpirationDate: now.Add(15897600 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	stored, ok, err := c.store.Get(clientID)
	if err != nil || !ok {
		t.Fatalf("stored token get: ok=%v err=%v", ok, err)
	}
	if diff := cmp.Diff(json.RawMessage(wantStored), stored); diff != "" {
		t.Fatalf("stored token (-want +got):\n%s", diff)
	}
}

// TestController_dispatchGet_usesUnlockFlag verifies that dispatch feeds handleGet the
// unlock-set refresh flag: a GET routed through handle refreshes only when the flag is on.
func TestController_dispatchGet_usesUnlockFlag(t *testing.T) {
	t.Parallel()
	newController := func(enable bool) (*Controller, *refreshRoundTripper) {
		c := newUnlockedController(t)
		c.enableRefreshToken = enable // as if unlocked with --enable-refresh
		now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		c.now = func() time.Time { return now }
		rt := &refreshRoundTripper{
			status: http.StatusOK,
			body:   `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":28800,"refresh_token_expires_in":15897600}`,
		}
		setClientTransport(c, rt)
		seedExpiredWithRefresh(t, c, "Iv1.x", now.Add(24*time.Hour))
		return c, rt
	}

	t.Run("enabled refreshes", func(t *testing.T) {
		t.Parallel()
		c, rt := newController(true)
		got, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.x"}`+"\n"))
		if !got.OK || !rt.called {
			t.Fatalf("a GET with refresh enabled must refresh; resp=%+v called=%v", got, rt.called)
		}
	})
	t.Run("disabled does not refresh", func(t *testing.T) {
		t.Parallel()
		c, rt := newController(false)
		got, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.x"}`+"\n"))
		if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
			t.Fatalf("GET with refresh disabled (-want +got):\n%s", diff)
		}
		if rt.called {
			t.Fatal("must not refresh when the unlock flag is off")
		}
	})
}

// TestController_handleGet_refreshTokenExpired verifies that an expired refresh token is
// not used and the request falls through to a not-found miss (no HTTP call).
func TestController_handleGet_refreshTokenExpired(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	rt := &refreshRoundTripper{status: http.StatusOK, body: `{}`}
	setClientTransport(c, rt)
	const clientID = "Iv1.refreshexp"
	seedExpiredWithRefresh(t, c, clientID, now.Add(-time.Hour)) // refresh token already expired

	got := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, true)
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
		t.Fatalf("expired refresh token (-want +got):\n%s", diff)
	}
	if rt.called {
		t.Fatal("must not call the refresh endpoint with an expired refresh token")
	}
}

// TestController_handleGet_refreshDisabled verifies that a valid refresh token is not used
// when refresh is disabled.
func TestController_handleGet_refreshDisabled(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	rt := &refreshRoundTripper{status: http.StatusOK, body: `{}`}
	setClientTransport(c, rt)
	const clientID = "Iv1.refreshoff"
	seedExpiredWithRefresh(t, c, clientID, now.Add(24*time.Hour))

	got := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, false)
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
		t.Fatalf("refresh disabled (-want +got):\n%s", diff)
	}
	if rt.called {
		t.Fatal("must not refresh when enableRefreshToken is false")
	}
}

// TestController_handleGet_refreshHTTPError verifies that a failing refresh request falls
// back to a not-found miss without corrupting the stored token.
func TestController_handleGet_refreshHTTPError(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	rt := &refreshRoundTripper{status: http.StatusInternalServerError, body: `{}`}
	setClientTransport(c, rt)
	const clientID = "Iv1.refresherr"
	seedExpiredWithRefresh(t, c, clientID, now.Add(24*time.Hour))

	got := c.handleGet(context.Background(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, true)
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
		t.Fatalf("refresh HTTP error (-want +got):\n%s", diff)
	}
	if !rt.called {
		t.Fatal("the refresh endpoint should have been called")
	}
	// The old token is left in place (not corrupted) on a refresh failure.
	if _, ok, err := c.store.Get(clientID); err != nil || !ok {
		t.Fatalf("stored token after failed refresh: ok=%v err=%v", ok, err)
	}
}
