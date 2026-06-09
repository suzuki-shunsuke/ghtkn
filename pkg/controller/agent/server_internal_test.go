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
)

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
	c.store = newDiskStore(testDataKey(t), t.TempDir())
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
	if err := newDiskStore(testDataKey(t), dir).Set("Iv1.abc", json.RawMessage(`{"access_token":"abc"}`)); err != nil {
		t.Fatal(err)
	}
	// Unlock the agent with a different key over the same directory.
	c := New()
	c.store = newDiskStore(make([]byte, dataKeyLen), dir)

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
	if err := newDiskStore(testDataKey(t), dir).Set("Iv1.old", json.RawMessage(`{"access_token":"x"}`)); err != nil {
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
