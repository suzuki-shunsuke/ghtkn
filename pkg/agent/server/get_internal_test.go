package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
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
func setClientTransport(c *Server, rt http.RoundTripper) {
	c.client = deviceflow.New(&deviceflow.Input{HTTPClient: &http.Client{Transport: rt}})
}

// roundTripFunc adapts a function to http.RoundTripper so a test can run a side effect on
// the refresh call (e.g. simulate a concurrent GET storing a fresh token).
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestServer_handleGet_pendingFlow verifies that a GET for a client with a device
// flow already in progress reports progress and echoes the one-time code, without
// touching the network.
func TestServer_handleGet_pendingFlow(t *testing.T) {
	t.Parallel()
	c := newUnlockedServer(t)
	const clientID = "Iv1.deviceflow"
	c.status[clientID] = &deviceFlowState{
		userCode:        "ABCD-1234",
		verificationURI: "https://github.com/login/device",
		expiresIn:       900,
	}
	got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, false)
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

// TestServer_handleGet_missNoFlow verifies that a plain GET (StartDeviceFlow false)
// for a client with no token and no in-progress flow is a not-found miss and does not
// start a device flow.
func TestServer_handleGet_missNoFlow(t *testing.T) {
	t.Parallel()
	c := newUnlockedServer(t)
	got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: "Iv1.notoken"}, false)
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
		t.Fatalf("plain GET miss (-want +got):\n%s", diff)
	}
	if len(c.status) != 0 {
		t.Fatalf("a plain GET must not start a flow; status has %d entries, want 0", len(c.status))
	}
}

// TestServer_handleGet_expired verifies that a plain GET for a token whose
// expiration date is in the past is a not-found miss. The controller reads the clock
// with time.Now, so the test runs in a synctest bubble, where time.Now is the bubble's
// fake clock and stands still for the whole test.
func TestServer_handleGet_expired(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		const clientID = "Iv1.expired"
		seeded := fmt.Sprintf(`{"access_token":"abc","expiration_date":"%s"}`, time.Now().Add(-time.Hour).Format(time.RFC3339))
		if err := c.store.Set(clientID, json.RawMessage(seeded)); err != nil {
			t.Fatal(err)
		}
		got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, false)
		if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
			t.Fatalf("expired GET (-want +got):\n%s", diff)
		}
	})
}

// TestServer_handleGet_minExpiration verifies that MinExpiration is honored: a token
// expiring 30m after now is returned for a 10m requirement but not for a 1h requirement.
func TestServer_handleGet_minExpiration(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		now := time.Now()
		const clientID = "Iv1.minexp"
		seeded := fmt.Sprintf(`{"access_token":"abc","expiration_date":"%s"}`, now.Add(30*time.Minute).Format(time.RFC3339))
		if err := c.store.Set(clientID, json.RawMessage(seeded)); err != nil {
			t.Fatal(err)
		}

		within := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID, MinExpiration: 10 * time.Minute}, false)
		if diff := cmp.Diff(&agentapi.Response{OK: true, Token: []byte(seeded)}, within); diff != "" {
			t.Fatalf("GET with 10m MinExpiration (-want +got):\n%s", diff)
		}

		beyond := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID, MinExpiration: time.Hour}, false)
		if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, beyond); diff != "" {
			t.Fatalf("GET with 1h MinExpiration (-want +got):\n%s", diff)
		}
	})
}

// TestServer_handleGet_awaitReturnsUnfreshToken verifies that a poll waiting for a
// device-flow result (AwaitDeviceFlow), once the flow has finished (no in-progress or
// failed marker), returns the stored token as is without the freshness check, so a
// freshly minted but already-past-expiry token is still handed back.
func TestServer_handleGet_awaitReturnsUnfreshToken(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		const clientID = "Iv1.await"
		// expiration_date is in the past, so a plain GET would treat it as a miss; the
		// await poll must still return it.
		seeded := fmt.Sprintf(`{"access_token":"minted","expiration_date":"%s"}`, time.Now().Add(-time.Hour).Format(time.RFC3339))
		if err := c.store.Set(clientID, json.RawMessage(seeded)); err != nil {
			t.Fatal(err)
		}
		got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID, AwaitDeviceFlow: true}, false)
		if diff := cmp.Diff(&agentapi.Response{OK: true, Token: []byte(seeded)}, got); diff != "" {
			t.Fatalf("await GET (-want +got):\n%s", diff)
		}
	})
}

// TestServer_handleGet_awaitNoToken verifies that a poll finding no stored token
// (the flow ended without minting one) reports not found.
func TestServer_handleGet_awaitNoToken(t *testing.T) {
	t.Parallel()
	c := newUnlockedServer(t)
	got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: "Iv1.none", AwaitDeviceFlow: true}, false)
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
		t.Fatalf("await GET without a token (-want +got):\n%s", diff)
	}
}

// seedExpiredWithRefresh stores an access token that expired an hour ago and whose
// refresh token is valid until refreshValidUntil. It must be called from within a
// synctest bubble, since the expiration is relative to the bubble's clock.
func seedExpiredWithRefresh(t *testing.T, c *Server, clientID string, refreshValidUntil time.Time) {
	t.Helper()
	seeded := fmt.Sprintf(`{"access_token":"old","expiration_date":"%s","refresh_token":"old-refresh","refresh_token_expiration_date":"%s"}`,
		time.Now().Add(-time.Hour).Format(time.RFC3339), refreshValidUntil.Format(time.RFC3339))
	if err := c.store.Set(clientID, json.RawMessage(seeded)); err != nil {
		t.Fatal(err)
	}
}

// TestServer_handleGet_refresh verifies that an expired access token with a valid
// refresh token is silently refreshed (no device flow), and the rotated token is stored.
func TestServer_handleGet_refresh(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		// The bubble's clock does not advance while the test goroutine runs, so the
		// expiration dates the controller computes are exactly now plus expires_in.
		now := time.Now()
		rt := &refreshRoundTripper{
			status: http.StatusOK,
			body:   `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":28800,"refresh_token_expires_in":15897600}`,
		}
		setClientTransport(c, rt)
		const clientID = "Iv1.refresh"
		seedExpiredWithRefresh(t, c, clientID, now.Add(24*time.Hour))

		got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, true)

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
		wantStored, err := json.Marshal(&storedToken{
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
	})
}

// TestServer_dispatchGet_usesUnlockFlag verifies that dispatch feeds handleGet the
// unlock-set refresh flag: a GET routed through handle refreshes only when the flag is on.
func TestServer_dispatchGet_usesUnlockFlag(t *testing.T) {
	t.Parallel()
	// newServer must be called from within a synctest bubble: the seeded token's
	// expiration is relative to the bubble's clock.
	newServer := func(t *testing.T, enable bool) (*Server, *refreshRoundTripper) {
		t.Helper()
		c := newUnlockedServer(t)
		c.enableRefreshToken = enable // as if unlocked with --enable-refresh
		rt := &refreshRoundTripper{
			status: http.StatusOK,
			body:   `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":28800,"refresh_token_expires_in":15897600}`,
		}
		setClientTransport(c, rt)
		seedExpiredWithRefresh(t, c, "Iv1.x", time.Now().Add(24*time.Hour))
		return c, rt
	}

	t.Run("enabled refreshes", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			c, rt := newServer(t, true)
			got, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.x"}`+"\n"))
			if !got.OK || !rt.called {
				t.Fatalf("a GET with refresh enabled must refresh; resp=%+v called=%v", got, rt.called)
			}
		})
	})
	t.Run("disabled does not refresh", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			c, rt := newServer(t, false)
			got, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.x"}`+"\n"))
			if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
				t.Fatalf("GET with refresh disabled (-want +got):\n%s", diff)
			}
			if rt.called {
				t.Fatal("must not refresh when the unlock flag is off")
			}
		})
	})
}

// TestServer_handleGet_refreshTokenExpired verifies that an expired refresh token is
// not used and the request falls through to a not-found miss (no HTTP call).
func TestServer_handleGet_refreshTokenExpired(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		rt := &refreshRoundTripper{status: http.StatusOK, body: `{}`}
		setClientTransport(c, rt)
		const clientID = "Iv1.refreshexp"
		seedExpiredWithRefresh(t, c, clientID, time.Now().Add(-time.Minute)) // refresh token already expired

		got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, true)
		if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
			t.Fatalf("expired refresh token (-want +got):\n%s", diff)
		}
		if rt.called {
			t.Fatal("must not call the refresh endpoint with an expired refresh token")
		}
	})
}

// TestServer_handleGet_refreshDisabled verifies that a valid refresh token is not used
// when refresh is disabled.
func TestServer_handleGet_refreshDisabled(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		rt := &refreshRoundTripper{status: http.StatusOK, body: `{}`}
		setClientTransport(c, rt)
		const clientID = "Iv1.refreshoff"
		seedExpiredWithRefresh(t, c, clientID, time.Now().Add(24*time.Hour))

		got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, false)
		if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
			t.Fatalf("refresh disabled (-want +got):\n%s", diff)
		}
		if rt.called {
			t.Fatal("must not refresh when enableRefreshToken is false")
		}
	})
}

// TestServer_handleGet_refreshHTTPError verifies that a failing refresh request falls
// back to a not-found miss without corrupting the stored token, and that because the
// refresh token was still valid, the response carries a security warning for the user.
func TestServer_handleGet_refreshHTTPError(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		rt := &refreshRoundTripper{status: http.StatusInternalServerError, body: `{}`}
		setClientTransport(c, rt)
		const clientID = "Iv1.refresherr"
		seedExpiredWithRefresh(t, c, clientID, time.Now().Add(24*time.Hour))

		got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, true)
		if got.Error != agentapi.RespNotFound {
			t.Fatalf("refresh HTTP error must be a not-found miss, got error %q", got.Error)
		}
		if got.Warning == "" {
			t.Fatal("a still-valid refresh token that failed to refresh must set an incident Warning")
		}
		if !rt.called {
			t.Fatal("the refresh endpoint should have been called")
		}
		// The old token is left in place (not corrupted) on a refresh failure.
		if _, ok, err := c.store.Get(clientID); err != nil || !ok {
			t.Fatalf("stored token after failed refresh: ok=%v err=%v", ok, err)
		}
	})
}

// TestServer_handleGet_refreshPeerAlreadyRefreshed verifies that when a refresh fails
// but a concurrent GET for the same client has already stored a fresh token (the likely
// cause with single-use rotating refresh tokens), the request serves that token silently
// instead of raising a false incident warning.
func TestServer_handleGet_refreshPeerAlreadyRefreshed(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		now := time.Now()
		const clientID = "Iv1.peerrefresh"
		seedExpiredWithRefresh(t, c, clientID, now.Add(24*time.Hour))

		// The refresh call fails, but as a side effect it stores a fresh, valid token, standing
		// in for a sibling GET that consumed the rotating refresh token first.
		fresh := fmt.Sprintf(`{"access_token":"peer-access","expiration_date":"%s","refresh_token":"peer-refresh","refresh_token_expiration_date":"%s"}`,
			now.Add(8*time.Hour).Format(time.RFC3339), now.Add(24*time.Hour).Format(time.RFC3339))
		var called bool
		setClientTransport(c, roundTripFunc(func(*http.Request) (*http.Response, error) {
			called = true
			if err := c.store.Set(clientID, json.RawMessage(fresh)); err != nil {
				t.Errorf("seed the peer-refreshed token: %v", err)
			}
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
		}))

		got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, true)
		if !called {
			t.Fatal("the refresh endpoint should have been called")
		}
		// The peer's fresh access token is served with no incident warning.
		//nolint:gosec // G117: serializing a token in a test to build the expected bytes.
		wantResp, err := json.Marshal(&struct {
			AccessToken    string    `json:"access_token"`
			ExpirationDate time.Time `json:"expiration_date"`
		}{AccessToken: "peer-access", ExpirationDate: now.Add(8 * time.Hour)})
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(&agentapi.Response{OK: true, Token: wantResp}, got); diff != "" {
			t.Fatalf("peer-refreshed GET (-want +got):\n%s", diff)
		}
	})
}

// TestServer_dropStaleAfterFailedStore verifies the recovery after a refresh whose
// store write failed: the stale token (still not valid for the request, so carrying a
// now-spent refresh token) is discarded so the next get re-authenticates, while a token
// that a concurrent refresh or device flow may have stored (valid for the request) is kept.
func TestServer_dropStaleAfterFailedStore(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		now := time.Now()

		// A stale (expired) token is dropped: the next get will re-authenticate instead of
		// trying its dead refresh token and raising a false incident warning.
		const staleID = "Iv1.stale"
		seedExpiredWithRefresh(t, c, staleID, now.Add(24*time.Hour))
		c.dropStaleAfterFailedStore(c.store, staleID, 0)
		if _, ok, _ := c.store.Get(staleID); ok {
			t.Fatal("a stale cached token must be dropped after a failed refresh store")
		}

		// A token still valid for the requirement is kept (stands in for one a concurrent
		// refresh or device flow stored in the meantime).
		const freshID = "Iv1.fresh"
		fresh := fmt.Sprintf(`{"access_token":"a","expiration_date":"%s"}`, now.Add(8*time.Hour).Format(time.RFC3339))
		if err := c.store.Set(freshID, json.RawMessage(fresh)); err != nil {
			t.Fatal(err)
		}
		c.dropStaleAfterFailedStore(c.store, freshID, time.Hour)
		if _, ok, _ := c.store.Get(freshID); !ok {
			t.Fatal("a token still valid for the requirement must be kept")
		}
	})
}

// TestServer_tokenValid_neverExpires verifies how a never-expiring token (zero
// expiration, from a GitHub App with token expiration disabled) is judged: it is valid
// for a normal min_expiration, but a min_expiration beyond the maximum token lifetime —
// the "regenerate regardless" signal ghtkn auth uses — makes it count as invalid so the
// token is regenerated.
func TestServer_tokenValid_neverExpires(t *testing.T) {
	t.Parallel()
	c := &Server{}
	// A token whose expiration_date is the zero time.
	raw := json.RawMessage(`{"access_token":"ghu_never"}`)

	if !c.tokenValid(raw, time.Hour) {
		t.Error("a never-expiring token must be valid for a normal min_expiration")
	}
	if !c.tokenValid(raw, maxTokenLifetime) {
		t.Error("a never-expiring token must be valid at exactly the max token lifetime")
	}
	if c.tokenValid(raw, maxTokenLifetime+time.Hour) {
		t.Error("a never-expiring token must count as invalid when min_expiration forces regeneration")
	}
}
