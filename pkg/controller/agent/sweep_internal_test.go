package agent

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestController_sweepExpiredTokens verifies that the sweep discards only tokens whose
// access token expired more than the TTL ago (a proxy for "unused for the TTL"), keeping
// still-valid tokens and those expired within the TTL.
func TestController_sweepExpiredTokens(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }

	seed := func(id string, expiration time.Time) {
		t.Helper()
		tok := fmt.Sprintf(`{"access_token":"x","expiration_date":"%s"}`, expiration.Format(time.RFC3339))
		if err := c.store.Set(id, json.RawMessage(tok)); err != nil {
			t.Fatal(err)
		}
	}
	seed("Iv1.stale", now.Add(-10*24*time.Hour)) // expired 10d ago: unused > 7d TTL
	seed("Iv1.recent", now.Add(-2*24*time.Hour)) // expired 2d ago: within the TTL
	seed("Iv1.fresh", now.Add(time.Hour))        // still valid

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
}
