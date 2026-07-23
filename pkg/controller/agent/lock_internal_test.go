package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// TestController_handle_lock verifies the LOCK command discards the in-memory data key:
// a subsequent GET reports locked, and STATUS reports locked but still initialized (the
// key file is kept). It drives everything through handle, so it also exercises the
// dispatch routing of LOCK.
func TestController_handle_lock(t *testing.T) {
	t.Parallel()
	c := New()
	c.keyFile = filepath.Join(t.TempDir(), "key")
	c.tokenDir = t.TempDir()

	unlock, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw"}`+"\n"))
	if !unlock.OK {
		t.Fatalf("UNLOCK failed: %+v", unlock)
	}
	lockResp, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"LOCK"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, Locked: true}, lockResp); diff != "" {
		t.Fatalf("LOCK (-want +got):\n%s", diff)
	}
	if c.tokenStore() != nil {
		t.Fatal("the store must be nil after LOCK")
	}
	getLocked, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.x"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespLocked}, getLocked); diff != "" {
		t.Fatalf("GET after lock (-want +got):\n%s", diff)
	}
	status, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"STATUS"}`+"\n"))
	if !status.Locked || !status.Initialized {
		t.Fatalf("STATUS after lock must be locked+initialized, got %+v", status)
	}
}

// TestController_handle_lock_reunlock verifies that LOCK keeps the key file, so a later
// UNLOCK with the same passphrase re-derives the data key and a token stored before the
// lock decrypts again.
func TestController_handle_lock_reunlock(t *testing.T) {
	t.Parallel()
	c := New()
	c.keyFile = filepath.Join(t.TempDir(), "key")
	c.tokenDir = t.TempDir()

	unlock, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw"}`+"\n"))
	if !unlock.OK {
		t.Fatalf("UNLOCK failed: %+v", unlock)
	}
	// Seed a never-expiring token (zero expiration is valid without a clock).
	if err := c.tokenStore().Set("Iv1.x", json.RawMessage(`{"access_token":"ghu_x"}`)); err != nil {
		t.Fatal(err)
	}
	if resp, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"LOCK"}`+"\n")); !resp.OK {
		t.Fatalf("LOCK failed: %+v", resp)
	}
	if resp, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw"}`+"\n")); !resp.OK {
		t.Fatalf("re-UNLOCK failed: %+v", resp)
	}
	get, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.x"}`+"\n"))
	if !get.OK || !strings.Contains(string(get.Token), "ghu_x") {
		t.Fatalf("GET after re-unlock must return the token, got %+v", get)
	}
}

// TestController_handleLock_idempotent verifies that locking an already-locked agent is a
// no-op success.
func TestController_handleLock_idempotent(t *testing.T) {
	t.Parallel()
	c := New() // locked: no store
	if diff := cmp.Diff(&agentapi.Response{OK: true, Locked: true}, c.handleLock()); diff != "" {
		t.Fatalf("LOCK on a locked agent (-want +got):\n%s", diff)
	}
}

// TestController_handleLock_stopsSweep verifies that LOCK stops the refresh-token sweep
// started at unlock: a token that the periodic sweep would discard on its next run is
// left in place because the sweep no longer runs after the lock. It mirrors
// TestController_startRefreshTokenSweep, which shows the same token IS swept without a
// lock.
func TestController_handleLock_stopsSweep(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := New()
		c.keyFile = filepath.Join(t.TempDir(), "key")
		c.tokenDir = t.TempDir()

		// Unlock with refresh enabled, which starts the sweep (bound to a cancelable ctx).
		unlock, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw","enable_refresh_token":true}`+"\n"))
		if !unlock.OK {
			t.Fatalf("UNLOCK --enable-refresh failed: %+v", unlock)
		}
		// Expired just over 6 days ago: within the 7d default TTL now, past it after a day.
		seedToken(t, c, "Iv1.aging", time.Now().Add(-6*24*time.Hour-time.Hour))
		synctest.Wait() // the immediate sweep has run; the token is still within the TTL
		tokenFile := filepath.Join(c.tokenDir, "Iv1.aging")
		if _, err := os.Stat(tokenFile); err != nil {
			t.Fatalf("the immediate sweep must keep a token still within the TTL: %v", err)
		}

		// Lock: this cancels the sweep. Wait for the goroutine to observe the cancellation.
		if resp := c.handleLock(); !resp.OK {
			t.Fatalf("LOCK failed: %+v", resp)
		}
		synctest.Wait()

		// A day passes: without the lock the periodic sweep would now discard the token, but
		// the sweep is stopped, so the token file survives.
		time.Sleep(refreshTokenSweepInterval)
		synctest.Wait()
		if _, err := os.Stat(tokenFile); err != nil {
			t.Fatalf("a locked agent must not sweep; the token file was removed: %v", err)
		}
	})
}
