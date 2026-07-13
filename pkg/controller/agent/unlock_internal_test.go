package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
)

// TestController_handle_unlock verifies the UNLOCK command loads the key and unlocks
// the agent.
func TestController_handle_unlock(t *testing.T) {
	t.Parallel()
	c := New()
	c.keyFile = filepath.Join(t.TempDir(), "key")
	c.tokenDir = t.TempDir()

	unlock, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, unlock); diff != "" {
		t.Fatalf("UNLOCK (-want +got):\n%s", diff)
	}
	// After unlock, GET works (returns not found rather than locked).
	get, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.x"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, get); diff != "" {
		t.Fatalf("GET after unlock (-want +got):\n%s", diff)
	}
}

// TestController_handle_unlock_enableRefresh verifies that UNLOCK binds refresh-token
// enablement to the passphrase moment: the flag is set at the first unlock, reported in
// the response and STATUS, and an already-unlocked re-unlock cannot flip it.
func TestController_handle_unlock_enableRefresh(t *testing.T) {
	t.Parallel()
	c := New()
	c.keyFile = filepath.Join(t.TempDir(), "key")
	c.tokenDir = t.TempDir()

	// Use a cancellable context: enabling refresh starts the sweep goroutine, which must
	// stop when the test ends.
	unlock, _ := c.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw","enable_refresh_token":true}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, RefreshTokenEnabled: true}, unlock); diff != "" {
		t.Fatalf("UNLOCK --enable-refresh (-want +got):\n%s", diff)
	}
	if !c.refreshEnabled() {
		t.Fatal("refresh must be enabled after unlock --enable-refresh")
	}
	// STATUS reports it.
	status, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"STATUS"}`+"\n"))
	if !status.RefreshTokenEnabled {
		t.Fatalf("STATUS must report refresh enabled, got %+v", status)
	}
	// An already-unlocked re-unlock without the flag must NOT flip it (the early-return
	// path never verifies the passphrase, so it must not change security state).
	reunlock, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, RefreshTokenEnabled: true}, reunlock); diff != "" {
		t.Fatalf("re-unlock (-want +got):\n%s", diff)
	}
	if !c.refreshEnabled() {
		t.Fatal("a re-unlock without the flag must not disable refresh")
	}
}

// TestController_handle_unlock_capsRefreshTokenTTL verifies the server clamps a
// refresh-token TTL larger than the six-month maximum. The CLI rejects such a value up
// front; this is the server-side backstop for any other client.
func TestController_handle_unlock_capsRefreshTokenTTL(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	c := New()
	c.keyFile = filepath.Join(t.TempDir(), "key")
	c.tokenDir = t.TempDir()

	// 200 days, over the six-month (180-day) cap. refresh_token_ttl is nanoseconds.
	overCap := (200 * 24 * time.Hour).Nanoseconds()
	req := fmt.Sprintf(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw","enable_refresh_token":true,"refresh_token_ttl":%d}`, overCap)
	unlock, _ := c.handle(ctx, strings.NewReader(req+"\n"))
	if !unlock.OK {
		t.Fatalf("unlock failed: %+v", unlock)
	}
	if c.refreshTokenTTL != MaxRefreshTokenTTL {
		t.Fatalf("refreshTokenTTL = %v, want capped to %v", c.refreshTokenTTL, MaxRefreshTokenTTL)
	}
}

// TestController_handle_unlock_orphanTokens verifies that unlocking with a freshly
// generated key warns when token files written under a previous key are still
// present (they can't be decrypted and will be re-minted).
func TestController_handle_unlock_orphanTokens(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A token left behind, encrypted under a previous key.
	if err := tokenstore.New(testDataKey(t), dir).Set("Iv1.old", json.RawMessage(`{"access_token":"x"}`)); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	c := New()
	c.logger = slog.New(slog.NewTextHandler(&buf, nil))
	c.keyFile = filepath.Join(t.TempDir(), "key") // absent: a new key is generated
	c.tokenDir = dir

	unlock, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, unlock); diff != "" {
		t.Fatalf("UNLOCK (-want +got):\n%s", diff)
	}
	if !strings.Contains(buf.String(), "predate the new agent key") {
		t.Fatalf("expected an orphan-token warning, got logs:\n%s", buf.String())
	}
}

// TestController_handle_unlock_stripsRefreshWhenDisabled verifies that unlocking with
// refresh disabled drops any stored refresh token (left by a previous refresh-enabled
// run) while keeping the access token.
func TestController_handle_unlock_stripsRefreshWhenDisabled(t *testing.T) {
	t.Parallel()
	keyFile := filepath.Join(t.TempDir(), "key")
	tokenDir := t.TempDir()

	// First unlock with refresh enabled, then seed a token carrying a refresh token.
	c1 := New()
	c1.keyFile = keyFile
	c1.tokenDir = tokenDir
	// Use a cancellable context: a refresh-enabled unlock starts the sweep goroutine,
	// which must stop when the test ends.
	if resp, _ := c1.handle(t.Context(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw","enable_refresh_token":true}`+"\n")); !resp.OK {
		t.Fatalf("first unlock failed: %+v", resp)
	}
	const seeded = `{"access_token":"ghu_a","expiration_date":"2999-01-01T00:00:00Z","refresh_token":"ghr_a","refresh_token_expiration_date":"2999-06-01T00:00:00Z"}`
	if err := c1.store.Set("Iv1.x", json.RawMessage(seeded)); err != nil {
		t.Fatal(err)
	}

	// Restart: a new controller unlocks over the same key/dir with refresh disabled.
	c2 := New()
	c2.keyFile = keyFile
	c2.tokenDir = tokenDir
	unlock, _ := c2.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw"}`+"\n"))
	if unlock.RefreshTokenEnabled {
		t.Fatalf("refresh must be disabled, got %+v", unlock)
	}

	raw, ok, err := c2.store.Get("Iv1.x")
	if err != nil || !ok {
		t.Fatalf("read the stripped token: ok=%v err=%v", ok, err)
	}
	var at struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(raw, &at); err != nil {
		t.Fatal(err)
	}
	if at.RefreshToken != "" {
		t.Fatalf("the refresh token must be stripped, got %q", at.RefreshToken)
	}
	if at.AccessToken != "ghu_a" {
		t.Fatalf("the access token must be preserved, got %q", at.AccessToken)
	}
}
