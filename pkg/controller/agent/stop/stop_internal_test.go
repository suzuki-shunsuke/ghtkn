package stop

import (
	"log/slog"
	"path/filepath"
	"testing"
)

// TestStop_notRunning verifies that stopping an agent that is not running succeeds
// (returns nil) rather than failing, mirroring 'systemctl stop'.
func TestStop_notRunning(t *testing.T) {
	t.Setenv("GHTKN_AGENT_SOCKET", filepath.Join(t.TempDir(), "absent.sock"))
	if err := New().Run(t.Context(), slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("Run with no agent running must return nil, got: %v", err)
	}
}
