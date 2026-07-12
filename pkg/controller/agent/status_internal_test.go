package agent

import (
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
)

// TestServe_status_locked verifies that a locked agent, served over a real socket,
// reports running and locked in response to STATUS.
func TestServe_status_locked(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "agent.sock")
	c := New() // starts locked: no store
	listener, err := listen(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go c.serve(t.Context(), listener, slog.New(slog.DiscardHandler)) //nolint:errcheck // serve returns nil once the listener is closed

	resp, err := agentapi.Send(t.Context(), path, &agentapi.Request{Command: agentapi.CommandStatus})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("resp.OK = false, error: %s", resp.Error)
	}
	if !resp.Locked {
		t.Fatal("a freshly started agent must report locked")
	}
}

// TestServe_status_unlocked verifies that an unlocked agent, served over a real
// socket, reports running, unlocked, and the cached token count in STATUS.
func TestServe_status_unlocked(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "agent.sock")
	c := New()
	// Unlock by installing a disk store before serving so the serve goroutine never
	// observes a concurrent write to c.store. Seed a token directly so STATUS reports
	// a non-zero count.
	c.store = tokenstore.New(testDataKey(t), t.TempDir())
	if err := c.store.Set("Iv1.x", json.RawMessage(`{"access_token":"abc"}`)); err != nil {
		t.Fatal(err)
	}
	listener, err := listen(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go c.serve(t.Context(), listener, slog.New(slog.DiscardHandler)) //nolint:errcheck // serve returns nil once the listener is closed

	resp, err := agentapi.Send(t.Context(), path, &agentapi.Request{Command: agentapi.CommandStatus})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK || resp.Locked {
		t.Fatalf("agent must be running and unlocked, got resp=%+v", resp)
	}
	if resp.Count != 1 {
		t.Fatalf("count = %d, want 1", resp.Count)
	}
}
