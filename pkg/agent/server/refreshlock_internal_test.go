package server

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
)

// TestServer_lockRefresh_canceled verifies that a GET waiting for another GET's refresh
// of the same client gives up when its context is canceled, and that the lock is per
// client: a different client is not blocked.
func TestServer_lockRefresh_canceled(t *testing.T) {
	t.Parallel()
	c := newUnlockedServer(t)
	lock, _, err := c.lockRefresh(t.Context(), "Iv1.a")
	if err != nil {
		t.Fatal(err)
	}
	defer lock.unlock()

	lockB, _, err := c.lockRefresh(t.Context(), "Iv1.b")
	if err != nil {
		t.Fatalf("another client must not be blocked: %v", err)
	}
	lockB.unlock()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := c.lockRefresh(ctx, "Iv1.a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("lockRefresh with a canceled context: got %v, want context.Canceled", err)
	}
}

// TestServer_lockRefresh_failureWarning verifies that lockRefresh reports a refresh failure
// only to a caller that was waiting when it happened, not to one that starts afterwards.
func TestServer_lockRefresh_failureWarning(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		const clientID = "Iv1.failurewarning"
		holder, _, err := c.lockRefresh(t.Context(), clientID)
		if err != nil {
			t.Fatal(err)
		}
		var waitWarning string
		done := make(chan struct{})
		go func() {
			defer close(done)
			lock, warning, err := c.lockRefresh(t.Context(), clientID)
			if err != nil {
				t.Error(err)
				return
			}
			waitWarning = warning
			lock.unlock()
		}()
		// Wait until the waiter is blocked on the lock.
		synctest.Wait()
		holder.markFailed("boom")
		holder.unlock()
		<-done
		if waitWarning != "boom" {
			t.Fatalf("waiting caller: warning %q, want %q", waitWarning, "boom")
		}

		later, warning, err := c.lockRefresh(t.Context(), clientID)
		if err != nil {
			t.Fatal(err)
		}
		defer later.unlock()
		if warning != "" {
			t.Fatalf("later caller: warning %q, want none", warning)
		}
	})
}

// TestServer_lockRefresh_failureThenSuccess verifies that a refresh succeeding after a
// failure clears the failure for the callers that waited across both, so they try the new
// refresh token instead of falling back with the stale warning.
func TestServer_lockRefresh_failureThenSuccess(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		const clientID = "Iv1.failurethensuccess"
		holder, _, err := c.lockRefresh(t.Context(), clientID)
		if err != nil {
			t.Fatal(err)
		}
		warning := "unset"
		done := make(chan struct{})
		go func() {
			defer close(done)
			lock, w, err := c.lockRefresh(t.Context(), clientID)
			if err != nil {
				t.Error(err)
				return
			}
			warning = w
			lock.unlock()
		}()
		synctest.Wait()
		holder.markFailed("boom")
		holder.markSucceeded()
		holder.unlock()
		<-done
		if warning != "" {
			t.Fatalf("waiting caller: warning %q, want none after a later success", warning)
		}
	})
}
