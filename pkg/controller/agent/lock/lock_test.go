package lock_test

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/lock"
)

// TestLock_notRunning verifies that locking an agent that is not running succeeds
// (returns nil) rather than failing: there is nothing unlocked to protect.
func TestLock_notRunning(t *testing.T) {
	t.Parallel()
	socket := filepath.Join(t.TempDir(), "absent.sock")
	c := lock.NewWithEnv(func(k string) string {
		if k == "GHTKN_AGENT_SOCKET" {
			return socket
		}
		return ""
	})
	if err := c.Run(t.Context(), slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("Run with no agent running must return nil, got: %v", err)
	}
}
