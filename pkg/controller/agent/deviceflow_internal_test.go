package agent

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
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
	// A token left from before the flow started: starting a flow no longer deletes it, so
	// it must still be there afterwards.
	if err := c.store.Set(clientID, json.RawMessage(`{"access_token":"old"}`)); err != nil {
		t.Fatal(err)
	}
	existing := &deviceFlowState{
		userCode:        "ABCD-1234",
		verificationURI: "https://github.com/login/device",
		expiresIn:       900,
	}
	c.status[clientID] = existing

	// deviceFlow returns a snapshot, so compare by value rather than pointer identity.
	state, err := c.startDeviceFlow(t.Context(), slog.New(slog.DiscardHandler), clientID, false)
	if err != nil {
		t.Fatalf("startDeviceFlow() error: %v", err)
	}
	if state == nil || *state != *existing {
		t.Fatalf("startDeviceFlow() = %+v, want the in-progress flow %+v", state, existing)
	}
	if rt.called {
		t.Fatal("a device code must not be requested when a flow is already in progress")
	}
	if _, ok, err := c.store.Get(clientID); err != nil || !ok {
		t.Fatalf("starting a flow must not delete the stored token: ok=%v err=%v", ok, err)
	}
}

// TestController_handleGet_awaitFailedFlow verifies that once a device flow has failed
// (its recorded state carries an error), a poll waiting on it (AwaitDeviceFlow) is told
// the flow failed, rather than being handed the pre-flow token that is no longer deleted.
func TestController_handleGet_awaitFailedFlow(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	const clientID = "Iv1.failed"
	// A still-valid token from before the flow. Without the failed marker, await would
	// return it as if the flow had produced it.
	seeded := `{"access_token":"old","expiration_date":"2999-01-01T00:00:00Z"}`
	if err := c.store.Set(clientID, json.RawMessage(seeded)); err != nil {
		t.Fatal(err)
	}
	c.status[clientID] = &deviceFlowState{errMsg: errMsgDeviceFlowFailed}

	got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID, AwaitDeviceFlow: true}, false)
	if got.OK || got.Error != errMsgDeviceFlowFailed {
		t.Fatalf("await of a failed flow = %+v, want the device-flow-failed error", got)
	}
}

// TestController_handleGet_failedFlowClearedByNonAwait verifies that a non-waiting GET
// discards a recorded failure so a later request is not stuck on it: a plain GET returns
// the still-valid pre-flow token, and the failed marker is gone.
func TestController_handleGet_failedFlowClearedByNonAwait(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	const clientID = "Iv1.failcleared"
	seeded := `{"access_token":"old","expiration_date":"2999-01-01T00:00:00Z"}`
	if err := c.store.Set(clientID, json.RawMessage(seeded)); err != nil {
		t.Fatal(err)
	}
	c.status[clientID] = &deviceFlowState{errMsg: errMsgDeviceFlowFailed}

	// A plain GET (not awaiting): the stale failure is dropped and the valid token served.
	got := c.handleGet(t.Context(), &agentapi.Request{ProtocolVersion: 1, Command: agentapi.CommandGet, ClientID: clientID}, false)
	if diff := cmp.Diff(&agentapi.Response{OK: true, Token: []byte(seeded)}, got); diff != "" {
		t.Fatalf("plain GET after a failed flow (-want +got):\n%s", diff)
	}
	if _, ok := c.status[clientID]; ok {
		t.Fatal("a non-waiting GET must clear the recorded failure")
	}
}
