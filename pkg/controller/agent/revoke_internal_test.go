package agent

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// TestController_handleRevoke verifies that REVOKE revokes the stored tokens' raw access
// tokens in one batch, reports no failures, and deletes the stored copies.
func TestController_handleRevoke(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	rev := &fakeRevoker{}
	c.revoker = rev
	if err := c.store.Set("Iv1.a", json.RawMessage(`{"access_token":"secret-a"}`)); err != nil {
		t.Fatal(err)
	}
	if err := c.store.Set("Iv1.b", json.RawMessage(`{"access_token":"secret-b"}`)); err != nil {
		t.Fatal(err)
	}
	got := c.handleRevoke(t.Context(), &agentapi.Request{Command: agentapi.CommandRevoke, ClientIDs: []string{"Iv1.a", "Iv1.b"}})
	if diff := cmp.Diff(&agentapi.Response{OK: true}, got); diff != "" {
		t.Fatalf("REVOKE (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"secret-a", "secret-b"}, rev.tokens); diff != "" {
		t.Fatalf("revoker received (-want +got):\n%s", diff)
	}
	for _, clientID := range []string{"Iv1.a", "Iv1.b"} {
		if _, ok, err := c.store.Get(clientID); err != nil || ok {
			t.Fatalf("token %s must be deleted after revoke: ok=%v err=%v", clientID, ok, err)
		}
	}
}

// TestController_handleRevoke_noToken verifies that a client with no stored token is
// silently skipped: only the seeded token is revoked and the response reports no
// failures.
func TestController_handleRevoke_noToken(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	rev := &fakeRevoker{}
	c.revoker = rev
	if err := c.store.Set("Iv1.a", json.RawMessage(`{"access_token":"secret-a"}`)); err != nil {
		t.Fatal(err)
	}
	got := c.handleRevoke(t.Context(), &agentapi.Request{Command: agentapi.CommandRevoke, ClientIDs: []string{"Iv1.a", "Iv1.absent"}})
	if diff := cmp.Diff(&agentapi.Response{OK: true}, got); diff != "" {
		t.Fatalf("REVOKE with a no-token client (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"secret-a"}, rev.tokens); diff != "" {
		t.Fatalf("revoker received (-want +got):\n%s", diff)
	}
}

// TestController_handleRevoke_revokeFails verifies that when the batch revoke fails, every
// attempted client ID is reported in RevokeFailed, no cleanup failure is reported, and the
// stored tokens are left in place.
func TestController_handleRevoke_revokeFails(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	c.revoker = &fakeRevoker{err: errors.New("boom")}
	if err := c.store.Set("Iv1.a", json.RawMessage(`{"access_token":"secret-a"}`)); err != nil {
		t.Fatal(err)
	}
	if err := c.store.Set("Iv1.b", json.RawMessage(`{"access_token":"secret-b"}`)); err != nil {
		t.Fatal(err)
	}
	got := c.handleRevoke(t.Context(), &agentapi.Request{Command: agentapi.CommandRevoke, ClientIDs: []string{"Iv1.a", "Iv1.b"}})
	if diff := cmp.Diff(&agentapi.Response{OK: true, RevokeFailed: []string{"Iv1.a", "Iv1.b"}}, got); diff != "" {
		t.Fatalf("REVOKE with a failing revoker (-want +got):\n%s", diff)
	}
	for _, clientID := range []string{"Iv1.a", "Iv1.b"} {
		if _, ok, err := c.store.Get(clientID); err != nil || !ok {
			t.Fatalf("token %s must remain stored on revoke failure: ok=%v err=%v", clientID, ok, err)
		}
	}
}

// TestController_handleRevoke_locked verifies that a locked agent refuses REVOKE.
func TestController_handleRevoke_locked(t *testing.T) {
	t.Parallel()
	c := New() // locked: no store
	got := c.handleRevoke(t.Context(), &agentapi.Request{Command: agentapi.CommandRevoke, ClientIDs: []string{"Iv1.x"}})
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespLocked}, got); diff != "" {
		t.Fatalf("REVOKE while locked (-want +got):\n%s", diff)
	}
}
