package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// TestController_handle_delete drives DELETE/GET against a seeded token and verifies
// that a deleted token is gone (GET reports not found) and that deleting an absent
// token succeeds.
func TestController_handle_delete(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)

	if err := c.store.Set("Iv1.abc", json.RawMessage(`{"access_token":"abc"}`)); err != nil {
		t.Fatal(err)
	}
	del, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"DELETE","client_id":"Iv1.abc"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, del); diff != "" {
		t.Fatalf("DELETE (-want +got):\n%s", diff)
	}
	get, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.abc"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, get); diff != "" {
		t.Fatalf("GET after DELETE (-want +got):\n%s", diff)
	}
	// Deleting an absent token is a no-op success.
	del2, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"DELETE","client_id":"Iv1.absent"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, del2); diff != "" {
		t.Fatalf("DELETE of an absent token (-want +got):\n%s", diff)
	}
}

// TestController_handle_delete_locked verifies that a locked agent refuses DELETE.
func TestController_handle_delete_locked(t *testing.T) {
	t.Parallel()
	c := New() // locked: no store
	del, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"DELETE","client_id":"Iv1.x"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespLocked}, del); diff != "" {
		t.Fatalf("DELETE while locked (-want +got):\n%s", diff)
	}
}
