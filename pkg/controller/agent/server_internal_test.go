package agent

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
)

// testDataKey returns a deterministic 32-byte key for tests.
func testDataKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

type handleTestCase struct {
	name     string
	requests []string
	want     []*agentapi.Response
}

var handleTestCases = []handleTestCase{ //nolint:gochecknoglobals // test fixture
	{
		name:     "status empty",
		requests: []string{`{"command":"STATUS"}`},
		want:     []*agentapi.Response{{OK: true}},
	},
	{
		name: "set then get",
		requests: []string{
			`{"command":"SET","client_id":"X","token":{"access_token":"abc"}}`,
			`{"command":"GET","client_id":"X"}`,
		},
		want: []*agentapi.Response{
			{OK: true},
			{OK: true, Token: []byte(`{"access_token":"abc"}`)},
		},
	},
	{
		name:     "get missing",
		requests: []string{`{"command":"GET","client_id":"missing"}`},
		want:     []*agentapi.Response{{Error: agentapi.RespNotFound}},
	},
	{
		name:     "get invalid client id",
		requests: []string{`{"command":"GET","client_id":"../escape"}`},
		want:     []*agentapi.Response{{Error: errMsgInvalidClientID}},
	},
	{
		name:     "set invalid client id",
		requests: []string{`{"command":"SET","client_id":"a/b","token":{}}`},
		want:     []*agentapi.Response{{Error: errMsgInvalidClientID}},
	},
	{
		name:     "unknown command",
		requests: []string{`{"command":"NOPE"}`},
		want:     []*agentapi.Response{{Error: errMsgUnknownCommand}},
	},
	{
		name:     "invalid json",
		requests: []string{`{not json`},
		want:     []*agentapi.Response{{Error: errMsgInvalidRequest}},
	},
	{
		name:     "empty request",
		requests: []string{""},
		want:     []*agentapi.Response{{Error: errMsgEmptyRequest}},
	},
	{
		name: "set then status counts",
		requests: []string{
			`{"command":"SET","client_id":"X","token":{"access_token":"abc"}}`,
			`{"command":"STATUS"}`,
		},
		want: []*agentapi.Response{
			{OK: true},
			{OK: true, Count: 1},
		},
	},
}

// newUnlockedController returns a controller with an in-place disk store, as if it
// had already been unlocked. The handle tests call it sequentially, so no locking
// around c.store is needed.
func newUnlockedController(t *testing.T) *Controller {
	t.Helper()
	c := New()
	c.store = tokenstore.New(testDataKey(t), t.TempDir())
	return c
}

func TestController_handle(t *testing.T) {
	t.Parallel()
	for _, d := range handleTestCases {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			c := newUnlockedController(t)
			for i, req := range d.requests {
				got, _ := c.handle(strings.NewReader(req + "\n"))
				if diff := cmp.Diff(d.want[i], got); diff != "" {
					t.Fatalf("request %d (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}

// TestController_handle_locked verifies that a locked agent refuses GET/SET and
// reports locked in STATUS.
func TestController_handle_locked(t *testing.T) {
	t.Parallel()
	c := New() // locked: no store

	get, _ := c.handle(strings.NewReader(`{"command":"GET","client_id":"Iv1.x"}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespLocked}, get); diff != "" {
		t.Fatalf("GET while locked (-want +got):\n%s", diff)
	}
	set, _ := c.handle(strings.NewReader(`{"command":"SET","client_id":"Iv1.x","token":{}}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespLocked}, set); diff != "" {
		t.Fatalf("SET while locked (-want +got):\n%s", diff)
	}
	status, _ := c.handle(strings.NewReader(`{"command":"STATUS"}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, Locked: true}, status); diff != "" {
		t.Fatalf("STATUS while locked (-want +got):\n%s", diff)
	}
}

// TestController_handle_disk drives GET/SET against a disk-backed encrypted store
// and verifies the token round-trips through encryption and disk persistence.
func TestController_handle_disk(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)

	set, _ := c.handle(strings.NewReader(`{"command":"SET","client_id":"Iv1.abc","token":{"access_token":"abc"}}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, set); diff != "" {
		t.Fatalf("SET (-want +got):\n%s", diff)
	}
	get, _ := c.handle(strings.NewReader(`{"command":"GET","client_id":"Iv1.abc"}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, Token: []byte(`{"access_token":"abc"}`)}, get); diff != "" {
		t.Fatalf("GET (-want +got):\n%s", diff)
	}
}

// TestController_handle_undecryptable verifies that a token persisted under a
// previous data key (which can't be decrypted after a key rotation) is treated as
// a cache miss (RespNotFound) so the client re-mints it, rather than a hard error.
func TestController_handle_undecryptable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Persist a token encrypted with one key.
	if err := tokenstore.New(testDataKey(t), dir).Set("Iv1.abc", json.RawMessage(`{"access_token":"abc"}`)); err != nil {
		t.Fatal(err)
	}
	// Unlock the agent with a different key over the same directory.
	c := New()
	c.store = tokenstore.New(make([]byte, 32), dir)

	get, _ := c.handle(strings.NewReader(`{"command":"GET","client_id":"Iv1.abc"}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, get); diff != "" {
		t.Fatalf("GET of an undecryptable token must be a miss (-want +got):\n%s", diff)
	}
}

// TestController_handle_unlock verifies the UNLOCK command loads the key and unlocks
// the agent.
func TestController_handle_unlock(t *testing.T) {
	t.Parallel()
	c := New()
	c.keyFile = filepath.Join(t.TempDir(), "key")
	c.tokenDir = t.TempDir()

	unlock, _ := c.handle(strings.NewReader(`{"command":"UNLOCK","passphrase":"pw"}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, unlock); diff != "" {
		t.Fatalf("UNLOCK (-want +got):\n%s", diff)
	}
	// After unlock, GET works (returns not found rather than locked).
	get, _ := c.handle(strings.NewReader(`{"command":"GET","client_id":"Iv1.x"}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, get); diff != "" {
		t.Fatalf("GET after unlock (-want +got):\n%s", diff)
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

	unlock, _ := c.handle(strings.NewReader(`{"command":"UNLOCK","passphrase":"pw"}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, unlock); diff != "" {
		t.Fatalf("UNLOCK (-want +got):\n%s", diff)
	}
	if !strings.Contains(buf.String(), "predate the new agent key") {
		t.Fatalf("expected an orphan-token warning, got logs:\n%s", buf.String())
	}
}

// TestController_handle_stop verifies the STOP command requests shutdown.
func TestController_handle_stop(t *testing.T) {
	t.Parallel()
	c := New()
	got, shutdown := c.handle(strings.NewReader(`{"command":"STOP"}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, got); diff != "" {
		t.Fatalf("response (-want +got):\n%s", diff)
	}
	if !shutdown {
		t.Fatal("STOP must request shutdown")
	}

	// Non-stop commands must not request shutdown.
	if _, shutdown := c.handle(strings.NewReader(`{"command":"STATUS"}` + "\n")); shutdown {
		t.Fatal("STATUS must not request shutdown")
	}
}

// TestServe_status_locked verifies that a locked agent, served over a real socket,
// reports running and locked in response to STATUS.
func TestServe_status_locked(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "agent.sock")
	c := New() // starts locked: no store
	listener, err := listen(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go c.serve(listener, slog.New(slog.DiscardHandler)) //nolint:errcheck // serve returns nil once the listener is closed

	resp, err := agentapi.Send(t.Context(), path, &agentapi.Request{Command: agentapi.CommandStatus})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("resp.OK = false, error: %s", resp.Error)
	}
	if !resp.Locked {
		t.Fatal("a freshly started agent must report locked")
	}
}

// TestServe_status_unlocked verifies that an unlocked agent, served over a real
// socket, reports running, unlocked, and the cached token count in STATUS.
func TestServe_status_unlocked(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "agent.sock")
	c := New()
	// Unlock by installing a disk store before serving so the serve goroutine never
	// observes a concurrent write to c.store.
	c.store = tokenstore.New(testDataKey(t), t.TempDir())
	listener, err := listen(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go c.serve(listener, slog.New(slog.DiscardHandler)) //nolint:errcheck // serve returns nil once the listener is closed

	if _, err := agentapi.Send(t.Context(), path, &agentapi.Request{Command: agentapi.CommandSet, ClientID: "Iv1.x", Token: []byte(`{"access_token":"abc"}`)}); err != nil {
		t.Fatal(err)
	}

	resp, err := agentapi.Send(t.Context(), path, &agentapi.Request{Command: agentapi.CommandStatus})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK || resp.Locked {
		t.Fatalf("agent must be running and unlocked, got resp=%+v", resp)
	}
	if resp.Count != 1 {
		t.Fatalf("count = %d, want 1", resp.Count)
	}
}
