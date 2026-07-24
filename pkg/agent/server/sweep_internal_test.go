package server

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"testing/synctest"
	"time"
)

// seedToken stores a token expiring at expiration. It must be called from within a
// synctest bubble when the expiration is relative to the bubble's clock.
func seedToken(t *testing.T, c *Server, id string, expiration time.Time) {
	t.Helper()
	tok := fmt.Sprintf(`{"access_token":"x","expiration_date":"%s"}`, expiration.Format(time.RFC3339))
	if err := c.store.Set(id, json.RawMessage(tok)); err != nil {
		t.Fatal(err)
	}
}

// TestServer_sweepExpiredTokens verifies that the sweep discards only tokens whose
// access token expired more than the TTL ago (a proxy for "unused for the TTL"), keeping
// still-valid tokens and those expired within the TTL.
func TestServer_sweepExpiredTokens(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		now := time.Now()

		seedToken(t, c, "Iv1.stale", now.Add(-10*24*time.Hour)) // expired 10d ago: unused > 7d TTL
		seedToken(t, c, "Iv1.recent", now.Add(-2*24*time.Hour)) // expired 2d ago: within the TTL
		seedToken(t, c, "Iv1.fresh", now.Add(time.Hour))        // still valid

		c.sweepExpiredTokens(c.store, 7*24*time.Hour)

		if _, ok, _ := c.store.Get("Iv1.stale"); ok {
			t.Fatal("a token unused past the TTL must be swept")
		}
		if _, ok, _ := c.store.Get("Iv1.recent"); !ok {
			t.Fatal("a token expired within the TTL must not be swept")
		}
		if _, ok, _ := c.store.Get("Iv1.fresh"); !ok {
			t.Fatal("a still-valid token must not be swept")
		}
	})
}

// TestServer_startRefreshTokenSweep verifies the periodic half of the sweep: the
// background job sweeps once immediately and then again every refreshTokenSweepInterval,
// and stops when the context is canceled. The bubble's fake clock makes the day between
// runs pass instantly, so the ticker loop is exercised rather than assumed.
func TestServer_startRefreshTokenSweep(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedServer(t)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		const ttl = 7 * 24 * time.Hour
		// Expired just over 6 days ago: within the 7d TTL now, past it a day later, so it
		// survives the immediate sweep and is discarded by the next tick.
		seedToken(t, c, "Iv1.aging", time.Now().Add(-6*24*time.Hour-time.Hour))

		c.startRefreshTokenSweep(ctx, c.store, ttl)
		synctest.Wait() // the immediate sweep has run
		if _, ok, _ := c.store.Get("Iv1.aging"); !ok {
			t.Fatal("the immediate sweep must keep a token still within the TTL")
		}

		time.Sleep(refreshTokenSweepInterval) // a day passes instantly
		synctest.Wait()                       // the tick's sweep has run
		if _, ok, _ := c.store.Get("Iv1.aging"); ok {
			t.Fatal("the periodic sweep must discard a token that has passed the TTL")
		}

		// Canceling the context stops the job; Test would report a deadlock if the
		// goroutine kept waiting on the ticker.
		cancel()
		synctest.Wait()
	})
}
