package agent

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
)

// TestController_startDeviceFlow_adoptsInProgressFlow verifies that a start for a client
// whose flow is already in progress adopts that flow without asking GitHub for a device
// code. handleGet reports a pending flow before it gets here, but a flow can start while
// the caller is reading the store or refreshing a token, and the device code fetched for
// the loser of that race would only expire unused.
func TestController_startDeviceFlow_adoptsInProgressFlow(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	// Any call to GitHub goes through this transport, so rt.called proves whether a device
	// code was requested.
	rt := &refreshRoundTripper{status: http.StatusOK, body: `{}`}
	setClientTransport(c, rt)
	const clientID = "Iv1.inflight"
	// A token left from before the flow started: only the flow that claims the slot may
	// delete it, so adopting must leave it alone.
	if err := c.store.Set(clientID, json.RawMessage(`{"access_token":"old"}`)); err != nil {
		t.Fatal(err)
	}
	existing := &deviceFlowState{
		userCode:        "ABCD-1234",
		verificationURI: "https://github.com/login/device",
		expiresIn:       900,
	}
	c.status[clientID] = existing

	state, err := c.startDeviceFlow(t.Context(), slog.New(slog.DiscardHandler), clientID, false)
	if err != nil {
		t.Fatalf("startDeviceFlow() error: %v", err)
	}
	if state != existing {
		t.Fatalf("startDeviceFlow() = %+v, want the in-progress flow %+v", state, existing)
	}
	if rt.called {
		t.Fatal("a device code must not be requested when a flow is already in progress")
	}
	if _, ok, err := c.store.Get(clientID); err != nil || !ok {
		t.Fatalf("adopting a flow must not delete the stored token: ok=%v err=%v", ok, err)
	}
}
