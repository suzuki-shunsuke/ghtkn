package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

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

	unlock, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"UNLOCK","passphrase":"pw","enable_refresh_token":true}`+"\n"))
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
