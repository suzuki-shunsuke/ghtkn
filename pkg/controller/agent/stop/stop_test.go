package stop_test

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/stop"
)

// TestStop_notRunning verifies that stopping an agent that is not running succeeds
// (returns nil) rather than failing, mirroring 'systemctl stop'.
func TestStop_notRunning(t *testing.T) {
	t.Parallel()
	socket := filepath.Join(t.TempDir(), "absent.sock")
	c := stop.NewWithEnv(func(k string) string {
		if k == "GHTKN_AGENT_SOCKET" {
			return socket
		}
		return ""
	})
	if err := c.Run(t.Context(), slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("Run with no agent running must return nil, got: %v", err)
	}
}
