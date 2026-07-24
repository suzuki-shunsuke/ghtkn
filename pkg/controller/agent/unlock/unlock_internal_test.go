package unlock

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// serveAgent starts a Unix-socket server that answers each request with handler, and
// returns a getEnv stub that points GHTKN_AGENT_SOCKET at it. Injecting the socket path
// through the Controller's getEnv (instead of t.Setenv) keeps the tests parallel-safe.
func serveAgent(t *testing.T, handler func(*agentapi.Request) *agentapi.Response) func(string) string {
	t.Helper()
	// A short dir keeps the socket path under the OS sun_path limit (t.TempDir embeds
	// the long test name).
	dir, err := os.MkdirTemp("", "gh") //nolint:usetesting // t.TempDir's path is too long for a unix socket
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "s.sock")

	lc := net.ListenConfig{}
	ln, err := lc.Listen(t.Context(), "unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			serveConn(conn, handler)
		}
	}()

	return func(k string) string {
		if k == "GHTKN_AGENT_SOCKET" {
			return socket
		}
		return ""
	}
}

// serveConn reads one newline-delimited request, answers it with handler, and writes the
// response back.
func serveConn(conn net.Conn, handler func(*agentapi.Request) *agentapi.Response) {
	defer conn.Close()
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return
	}
	req := &agentapi.Request{}
	if err := json.Unmarshal(line, req); err != nil {
		return
	}
	b, err := json.Marshal(handler(req))
	if err != nil {
		return
	}
	_, _ = conn.Write(append(b, '\n'))
}

// pendingHandler answers STATUS with locked, the first UNLOCK with RefreshTokenRemovalPending,
// and any UNLOCK carrying ConfirmRefreshTokenRemoval with OK. It records every UNLOCK seen.
func pendingHandler(mu *sync.Mutex, unlocks *[]*agentapi.Request) func(*agentapi.Request) *agentapi.Response {
	return func(req *agentapi.Request) *agentapi.Response {
		switch req.Command {
		case agentapi.CommandStatus:
			return &agentapi.Response{OK: true, Locked: true, Initialized: true}
		case agentapi.CommandUnlock:
			mu.Lock()
			*unlocks = append(*unlocks, req)
			mu.Unlock()
			if req.ConfirmRefreshTokenRemoval {
				return &agentapi.Response{OK: true}
			}
			return &agentapi.Response{RefreshTokenRemovalPending: true, Error: "confirm"}
		default:
			return &agentapi.Response{Error: "unexpected command"}
		}
	}
}

// TestController_Run_enableRefresh verifies that --enable-refresh reaches the wire: the
// UNLOCK request the client sends carries EnableRefreshToken.
func TestController_Run_enableRefresh(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var unlockReq *agentapi.Request
	getEnv := serveAgent(t, func(req *agentapi.Request) *agentapi.Response {
		switch req.Command {
		case agentapi.CommandStatus:
			return &agentapi.Response{OK: true, Locked: true, Initialized: true}
		case agentapi.CommandUnlock:
			mu.Lock()
			unlockReq = req
			mu.Unlock()
			return &agentapi.Response{OK: true, RefreshTokenEnabled: req.EnableRefreshToken}
		default:
			return &agentapi.Response{Error: "unexpected command"}
		}
	})

	c := &Controller{
		readPassphrase: func(string) ([]byte, error) { return []byte("pw"), nil },
		getEnv:         getEnv,
	}
	if err := c.Run(t.Context(), slog.New(slog.DiscardHandler), true, 0); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if unlockReq == nil {
		t.Fatal("no UNLOCK request was received")
	}
	if !unlockReq.EnableRefreshToken {
		t.Fatal("the UNLOCK request must carry EnableRefreshToken=true")
	}
}

// TestController_Run_confirmRefreshRemoval verifies that when the agent reports
// RefreshTokenRemovalPending and the user confirms, the client re-sends the unlock with
// ConfirmRefreshTokenRemoval set.
func TestController_Run_confirmRefreshRemoval(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var unlocks []*agentapi.Request
	getEnv := serveAgent(t, pendingHandler(&mu, &unlocks))

	c := &Controller{
		readPassphrase: func(string) ([]byte, error) { return []byte("pw"), nil },
		confirm:        func(string) (bool, error) { return true, nil },
		getEnv:         getEnv,
	}
	if err := c.Run(t.Context(), slog.New(slog.DiscardHandler), false, 0); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(unlocks) != 2 {
		t.Fatalf("expected 2 UNLOCK requests (initial + confirmed), got %d", len(unlocks))
	}
	if unlocks[0].ConfirmRefreshTokenRemoval {
		t.Fatal("the first UNLOCK must not carry the confirmation")
	}
	if !unlocks[1].ConfirmRefreshTokenRemoval {
		t.Fatal("the re-sent UNLOCK must carry ConfirmRefreshTokenRemoval=true")
	}
}

// TestController_Run_declineRefreshRemoval verifies that declining the prompt aborts
// without re-sending the unlock and without an error (the agent stays locked).
func TestController_Run_declineRefreshRemoval(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var unlocks []*agentapi.Request
	getEnv := serveAgent(t, pendingHandler(&mu, &unlocks))

	c := &Controller{
		readPassphrase: func(string) ([]byte, error) { return []byte("pw"), nil },
		confirm:        func(string) (bool, error) { return false, nil },
		getEnv:         getEnv,
	}
	if err := c.Run(t.Context(), slog.New(slog.DiscardHandler), false, 0); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(unlocks) != 1 {
		t.Fatalf("expected only the initial UNLOCK after declining, got %d", len(unlocks))
	}
}
