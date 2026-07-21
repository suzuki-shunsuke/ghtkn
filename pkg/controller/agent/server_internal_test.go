package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

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
		requests: []string{`{"protocol_version":1,"command":"STATUS"}`},
		want:     []*agentapi.Response{{OK: true}},
	},
	{
		name:     "get missing",
		requests: []string{`{"protocol_version":1,"command":"GET","client_id":"missing"}`},
		want:     []*agentapi.Response{{Error: agentapi.RespNotFound}},
	},
	{
		name:     "get invalid client id",
		requests: []string{`{"protocol_version":1,"command":"GET","client_id":"../escape"}`},
		want:     []*agentapi.Response{{Error: errMsgInvalidClientID}},
	},
	{
		name:     "unknown command",
		requests: []string{`{"protocol_version":1,"command":"NOPE"}`},
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
				got, _ := c.handle(context.Background(), strings.NewReader(req+"\n"))
				if diff := cmp.Diff(d.want[i], got); diff != "" {
					t.Fatalf("request %d (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}

// TestController_handle_getSeeded verifies that a token seeded directly into the store
// is returned by GET.
func TestController_handle_getSeeded(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	const seeded = `{"access_token":"abc","expiration_date":"2999-01-01T00:00:00Z"}`
	if err := c.store.Set("X", json.RawMessage(seeded)); err != nil {
		t.Fatal(err)
	}
	got, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"X"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, Token: []byte(seeded)}, got); diff != "" {
		t.Fatalf("GET (-want +got):\n%s", diff)
	}
}

// TestController_handle_statusCounts verifies that STATUS reports the number of cached
// tokens.
func TestController_handle_statusCounts(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	if err := c.store.Set("X", json.RawMessage(`{"access_token":"abc"}`)); err != nil {
		t.Fatal(err)
	}
	got, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"STATUS"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, Count: 1}, got); diff != "" {
		t.Fatalf("STATUS (-want +got):\n%s", diff)
	}
}

// TestController_handle_obsoleteClient verifies that a request below the minimum
// supported protocol version is rejected as an obsolete client before any dispatch.
// MinProtocolVersion is currently 0, so a negative version stands in for a
// hypothetical dropped-support version.
func TestController_handle_obsoleteClient(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	tooOld := agentapi.MinProtocolVersion - 1
	got, _ := c.handle(context.Background(), strings.NewReader(fmt.Sprintf(`{"protocol_version":%d,"command":"GET","client_id":"Iv1.x"}`, tooOld)+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespObsoleteClient}, got); diff != "" {
		t.Fatalf("obsolete client (-want +got):\n%s", diff)
	}
}

// TestController_handle_legacySetGet verifies that a legacy (protocol version 0,
// i.e. no protocol_version field) client is served in compatibility mode: it stores
// a self-minted token with SET and reads it back with GET, instead of being rejected.
func TestController_handle_legacySetGet(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	const token = `{"access_token":"abc","expiration_date":"2999-01-01T00:00:00Z"}`
	set, _ := c.handle(context.Background(), strings.NewReader(`{"command":"SET","client_id":"Iv1.x","token":`+token+`}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, set); diff != "" {
		t.Fatalf("legacy SET (-want +got):\n%s", diff)
	}
	get, _ := c.handle(context.Background(), strings.NewReader(`{"command":"GET","client_id":"Iv1.x"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, Token: []byte(token)}, get); diff != "" {
		t.Fatalf("legacy GET (-want +got):\n%s", diff)
	}
}

// TestController_handle_setRejectedForCurrentClient verifies that SET is refused for a
// non-legacy (protocol version 1) client: the server owns the token lifecycle, so a
// client must not be able to overwrite the stored token with a self-pushed one.
func TestController_handle_setRejectedForCurrentClient(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	const token = `{"access_token":"abc","expiration_date":"2999-01-01T00:00:00Z"}`
	set, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"SET","client_id":"Iv1.x","token":`+token+`}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: errMsgSetNotAllowed}, set); diff != "" {
		t.Fatalf("current-client SET (-want +got):\n%s", diff)
	}
	// Nothing was stored: a subsequent GET is a miss.
	get, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.x"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, get); diff != "" {
		t.Fatalf("GET after rejected SET (-want +got):\n%s", diff)
	}
}

// TestController_handle_legacyGetMissing verifies that a legacy GET miss returns
// RespNotFound (the agent never starts a device flow for a legacy client, so the
// client re-mints the token itself and stores it with SET).
func TestController_handle_legacyGetMissing(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	got, _ := c.handle(context.Background(), strings.NewReader(`{"command":"GET","client_id":"missing"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
		t.Fatalf("legacy GET miss (-want +got):\n%s", diff)
	}
}

// TestController_handle_legacyNoRefresh verifies that even when the agent was unlocked
// with refresh enabled and holds an expired token with a valid refresh token, a legacy
// (version 0) GET does not refresh it: refresh is gated off by protocol version, so the
// legacy client gets a cache miss and re-mints, matching the old behavior where refresh
// is unsupported. The fake refresh endpoint would return a new token if it were called,
// so a not-found response together with rt.called == false proves refresh was skipped.
// The version-1 refresh path itself is covered by TestController_handleGet_refresh.
func TestController_handle_legacyNoRefresh(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		c := newUnlockedController(t)
		c.enableRefreshToken = true
		rt := &refreshRoundTripper{
			status: http.StatusOK,
			body:   `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":28800,"refresh_token_expires_in":15897600}`,
		}
		setClientTransport(c, rt)
		const clientID = "Iv1.legacy"
		seedExpiredWithRefresh(t, c, clientID, time.Now().Add(24*time.Hour))

		// Legacy GET: no protocol_version field (decodes to 0).
		got, _ := c.handle(context.Background(), strings.NewReader(`{"command":"GET","client_id":"`+clientID+`"}`+"\n"))
		if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, got); diff != "" {
			t.Fatalf("legacy GET must not refresh (-want +got):\n%s", diff)
		}
		if rt.called {
			t.Fatal("a legacy GET must not attempt a refresh, but the refresh endpoint was called")
		}
	})
}

// TestController_handle_obsoleteAgent verifies that a request with a protocol version
// newer than the agent's (the agent is out of date) is rejected before any dispatch.
func TestController_handle_obsoleteAgent(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)
	newer := agentapi.ProtocolVersion + 1
	got, _ := c.handle(context.Background(), strings.NewReader(fmt.Sprintf(`{"protocol_version":%d,"command":"GET","client_id":"Iv1.x"}`, newer)+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespObsoleteAgent}, got); diff != "" {
		t.Fatalf("obsolete agent (-want +got):\n%s", diff)
	}
}

// TestController_handle_locked verifies that a locked agent refuses GET and reports
// locked in STATUS.
func TestController_handle_locked(t *testing.T) {
	t.Parallel()
	c := New() // locked: no store

	get, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.x"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespLocked}, get); diff != "" {
		t.Fatalf("GET while locked (-want +got):\n%s", diff)
	}
	status, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"STATUS"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, Locked: true}, status); diff != "" {
		t.Fatalf("STATUS while locked (-want +got):\n%s", diff)
	}
}

// TestController_handle_disk drives GET against a disk-backed encrypted store and
// verifies the token round-trips through encryption and disk persistence.
func TestController_handle_disk(t *testing.T) {
	t.Parallel()
	c := newUnlockedController(t)

	const seeded = `{"access_token":"abc","expiration_date":"2999-01-01T00:00:00Z"}`
	if err := c.store.Set("Iv1.abc", json.RawMessage(seeded)); err != nil {
		t.Fatal(err)
	}
	get, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.abc"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, Token: []byte(seeded)}, get); diff != "" {
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

	get, _ := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"GET","client_id":"Iv1.abc"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{Error: agentapi.RespNotFound}, get); diff != "" {
		t.Fatalf("GET of an undecryptable token must be a miss (-want +got):\n%s", diff)
	}
}

// TestController_handle_stop verifies the STOP command requests shutdown.
func TestController_handle_stop(t *testing.T) {
	t.Parallel()
	c := New()
	got, shutdown := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"STOP"}`+"\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, got); diff != "" {
		t.Fatalf("response (-want +got):\n%s", diff)
	}
	if !shutdown {
		t.Fatal("STOP must request shutdown")
	}

	// Non-stop commands must not request shutdown.
	if _, shutdown := c.handle(context.Background(), strings.NewReader(`{"protocol_version":1,"command":"STATUS"}`+"\n")); shutdown {
		t.Fatal("STATUS must not request shutdown")
	}
}

// fakeRevoker is a test double for the revoker interface. It records the tokens it was
// asked to revoke and returns a configurable error.
type fakeRevoker struct {
	tokens []string
	err    error
}

func (f *fakeRevoker) Revoke(_ context.Context, tokens []string) error {
	f.tokens = append(f.tokens, tokens...)
	return f.err
}

// TestServe_oversized verifies that a request larger than maxRequestBytes with no newline
// is capped and answered with an error, rather than buffered without limit or left to hang.
// The name is kept short so the socket path stays under the OS sun_path limit.
func TestServe_oversized(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "agent.sock")
	c := New()
	listener, err := listen(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go c.serve(t.Context(), listener, slog.New(slog.DiscardHandler)) //nolint:errcheck // serve returns nil once the listener is closed

	var d net.Dialer
	conn, err := d.DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	// More than maxRequestBytes and no newline. The server stops reading at the cap, so a
	// short write is expected; write in a goroutine so this test never blocks on it.
	oversized := bytes.Repeat([]byte("a"), maxRequestBytes+4096)
	go conn.Write(oversized) //nolint:errcheck

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		t.Fatalf("read the response: %v", err)
	}
	resp := &agentapi.Response{}
	if err := json.Unmarshal(bytes.TrimSpace(line), resp); err != nil {
		t.Fatalf("unmarshal the response %q: %v", line, err)
	}
	if resp.Error == "" {
		t.Fatalf("an oversized request must be rejected with an error, got %+v", resp)
	}
}
