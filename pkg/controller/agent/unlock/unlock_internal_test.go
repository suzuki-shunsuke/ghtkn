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

// TestController_Run_enableRefresh verifies that --enable-refresh reaches the wire: the
// UNLOCK request the client sends carries EnableRefreshToken.
func TestController_Run_enableRefresh(t *testing.T) {
	// A short dir keeps the socket path under the OS sun_path limit (t.TempDir embeds
	// the long test name).
	dir, err := os.MkdirTemp("", "gh") //nolint:usetesting // t.TempDir's path is too long for a unix socket
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "s.sock")
	t.Setenv("GHTKN_AGENT_SOCKET", socket)

	lc := net.ListenConfig{}
	ln, err := lc.Listen(t.Context(), "unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	var mu sync.Mutex
	var unlockReq *agentapi.Request
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			serveOne(conn, &mu, &unlockReq)
		}
	}()

	c := &Controller{readPassphrase: func(string) ([]byte, error) { return []byte("pw"), nil }}
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

// serveOne answers one request: STATUS reports locked (so Run prompts and unlocks), and
// UNLOCK is captured and echoed back with its refresh state.
func serveOne(conn net.Conn, mu *sync.Mutex, unlockReq **agentapi.Request) {
	defer conn.Close()
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return
	}
	req := &agentapi.Request{}
	if err := json.Unmarshal(line, req); err != nil {
		return
	}
	var resp *agentapi.Response
	switch req.Command {
	case agentapi.CommandStatus:
		resp = &agentapi.Response{OK: true, Locked: true, Initialized: true}
	case agentapi.CommandUnlock:
		mu.Lock()
		*unlockReq = req
		mu.Unlock()
		resp = &agentapi.Response{OK: true, RefreshTokenEnabled: req.EnableRefreshToken}
	default:
		resp = &agentapi.Response{Error: "unexpected command"}
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = conn.Write(append(b, '\n'))
}
