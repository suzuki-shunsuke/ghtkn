package agent

import (
	"log/slog"
	"path/filepath"
	"testing"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

func TestQueryStatus_notRunning(t *testing.T) {
	t.Parallel()
	resp, running, err := queryStatus(t.Context(), filepath.Join(t.TempDir(), "absent.sock"))
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("queryStatus must report not running when the socket is absent")
	}
	if resp != nil {
		t.Fatalf("resp = %+v, want nil", resp)
	}
}

func TestQueryStatus_locked(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "agent.sock")
	c := New() // starts locked: no store
	listener, err := listen(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go c.serve(listener, slog.New(slog.DiscardHandler)) //nolint:errcheck // serve returns nil once the listener is closed

	resp, running, err := queryStatus(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !running {
		t.Fatal("queryStatus must report running")
	}
	if !resp.Locked {
		t.Fatal("a freshly started agent must report locked")
	}
}

func TestQueryStatus_unlocked(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "agent.sock")
	c := New()
	// Unlock by installing a disk store before serving so the serve goroutine never
	// observes a concurrent write to c.store.
	c.store = newDiskStore(testDataKey(t), t.TempDir())
	listener, err := listen(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go c.serve(listener, slog.New(slog.DiscardHandler)) //nolint:errcheck // serve returns nil once the listener is closed

	if _, err := agentapi.Send(t.Context(), path, &agentapi.Request{Command: agentapi.CommandSet, ClientID: "Iv1.x", Token: []byte(`{"access_token":"abc"}`)}); err != nil {
		t.Fatal(err)
	}

	resp, running, err := queryStatus(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !running || resp.Locked {
		t.Fatalf("agent must be running and unlocked, got running=%v resp=%+v", running, resp)
	}
	if resp.Count != 1 {
		t.Fatalf("count = %d, want 1", resp.Count)
	}
}
