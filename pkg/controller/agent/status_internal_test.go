package agent

import (
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
)

// TestController_handleStatus_refreshTTL verifies that STATUS reports the refresh-token
// TTL only when the agent is unlocked with refresh enabled (a locked agent has refresh
// off, so it reports neither).
func TestController_handleStatus_refreshTTL(t *testing.T) {
	t.Parallel()
	c := New()
	c.store = tokenstore.New(testDataKey(t), t.TempDir()) // unlocked
	c.enableRefreshToken = true
	c.refreshTokenTTL = 3 * 24 * time.Hour
	if resp := c.handleStatus(); !resp.RefreshTokenEnabled || resp.RefreshTokenTTL != 3*24*time.Hour {
		t.Fatalf("unlocked+refresh STATUS must report the TTL, got %+v", resp)
	}

	locked := New() // locked: no store, refresh off
	if resp := locked.handleStatus(); resp.RefreshTokenEnabled || resp.RefreshTokenTTL != 0 {
		t.Fatalf("a locked agent must not report a refresh TTL, got %+v", resp)
	}
}

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

// TestServe_protoVersion verifies that a response served over the socket carries this
// agent's protocol version. A client that needs the server-owned token lifecycle
// refuses an agent whose responses lack it (agentapi.ErrObsoleteAgent), so forgetting
// to stamp it would make every current client treat this agent as obsolete. The test
// name is kept short so the socket path stays under the platform's sun_path limit.
func TestServe_protoVersion(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "a.sock")
	c := New()
	listener, err := listen(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go c.serve(t.Context(), listener, slog.New(slog.DiscardHandler)) //nolint:errcheck // serve returns nil once the listener is closed

	// STATUS on a locked agent: even a response built before any dispatch must be
	// stamped, so the version does not depend on which command was served.
	resp, err := agentapi.Send(t.Context(), path, &agentapi.Request{Command: agentapi.CommandStatus})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ProtocolVersion != agentapi.ProtocolVersion {
		t.Fatalf("resp.ProtocolVersion = %d, want %d", resp.ProtocolVersion, agentapi.ProtocolVersion)
	}
}
