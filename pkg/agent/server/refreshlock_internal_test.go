package server

import (
	"context"
	"errors"
	"testing"
)

// TestServer_lockRefresh_canceled verifies that a GET waiting for another GET's refresh
// of the same client gives up when its context is canceled, and that the lock is per
// client: a different client is not blocked.
func TestServer_lockRefresh_canceled(t *testing.T) {
	t.Parallel()
	c := newUnlockedServer(t)
	lock, err := c.lockRefresh(t.Context(), "Iv1.a")
	if err != nil {
		t.Fatal(err)
	}
	defer lock.unlock()

	lockB, err := c.lockRefresh(t.Context(), "Iv1.b")
	if err != nil {
		t.Fatalf("another client must not be blocked: %v", err)
	}
	lockB.unlock()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.lockRefresh(ctx, "Iv1.a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("lockRefresh with a canceled context: got %v, want context.Canceled", err)
	}
}
