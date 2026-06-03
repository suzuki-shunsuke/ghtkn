package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// Error messages returned to clients in the response.
const (
	errMsgEmptyRequest    = "empty request"
	errMsgInvalidRequest  = "invalid request"
	errMsgReadRequest     = "read the request"
	errMsgUnknownCommand  = "unknown command"
	errMsgInvalidClientID = "invalid client id"
	errMsgGet             = "get the token"
	errMsgSet             = "set the token"
	errMsgUnlock          = "unlock the agent"
)

// serve accepts connections until the listener is closed and handles each one.
// It returns nil when the listener is closed (e.g. on shutdown).
func (c *Controller) serve(listener net.Listener, logger *slog.Logger) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err //nolint:wrapcheck
		}
		go c.handleConn(conn, logger)
	}
}

// handleConn reads a single request from conn, processes it, writes the response,
// and closes the connection. Each connection serves exactly one request.
// When the request asks the agent to stop, the shutdown is triggered only after
// the response has been written so the client always receives the acknowledgment.
func (c *Controller) handleConn(conn net.Conn, logger *slog.Logger) {
	defer conn.Close()
	resp, shutdown := c.handle(conn)
	b, err := json.Marshal(resp)
	if err != nil {
		logger.Error("marshal the agent response", "error", err)
		return
	}
	if _, err := conn.Write(append(b, '\n')); err != nil {
		logger.Error("write the agent response", "error", err)
	}
	if shutdown && c.shutdown != nil {
		c.shutdown()
	}
}

// handle reads and processes one request, returning the response to send and
// whether the agent should shut down afterwards.
func (c *Controller) handle(r io.Reader) (*agentapi.Response, bool) {
	line, err := bufio.NewReader(r).ReadBytes('\n')
	// ReadBytes returns io.EOF together with the data when there is no trailing
	// newline, so a non-empty line is still valid in that case.
	if err != nil && !errors.Is(err, io.EOF) {
		return &agentapi.Response{Error: errMsgReadRequest}, false
	}
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return &agentapi.Response{Error: errMsgEmptyRequest}, false
	}
	req := &agentapi.Request{}
	if err := json.Unmarshal(line, req); err != nil {
		return &agentapi.Response{Error: errMsgInvalidRequest}, false
	}
	return c.dispatch(req)
}

// dispatch routes a request to the matching handler.
// The second return value reports whether the agent should shut down.
func (c *Controller) dispatch(req *agentapi.Request) (*agentapi.Response, bool) {
	switch req.Command {
	case agentapi.CommandGet:
		return c.handleGet(req), false
	case agentapi.CommandSet:
		return c.handleSet(req), false
	case agentapi.CommandStatus:
		return c.handleStatus(), false
	case agentapi.CommandUnlock:
		return c.handleUnlock(req), false
	case agentapi.CommandStop:
		return &agentapi.Response{OK: true}, true
	default:
		return &agentapi.Response{Error: errMsgUnknownCommand}, false
	}
}

// handleGet returns the cached token for the request's client ID.
func (c *Controller) handleGet(req *agentapi.Request) *agentapi.Response {
	st := c.tokenStore()
	if st == nil {
		return &agentapi.Response{Error: agentapi.RespLocked}
	}
	token, ok, err := st.Get(req.ClientID)
	switch {
	case errors.Is(err, errInvalidClientID):
		return &agentapi.Response{Error: errMsgInvalidClientID}
	case errors.Is(err, errDecryptToken):
		// A token persisted under a previous data key (e.g. after a key
		// rotation) can't be decrypted. Treat it as a cache miss so the client
		// re-mints the token via the device flow and overwrites the stale file,
		// instead of failing permanently with an opaque error.
		if c.logger != nil {
			slogerr.WithError(c.logger, err).Warn("discard an undecryptable cached token; it will be re-minted", "client_id", req.ClientID)
		}
		return &agentapi.Response{Error: agentapi.RespNotFound}
	case err != nil:
		return &agentapi.Response{Error: fmt.Sprintf("%s: %s", errMsgGet, err)}
	case !ok:
		return &agentapi.Response{Error: agentapi.RespNotFound}
	}
	return &agentapi.Response{OK: true, Token: token}
}

// handleSet stores the request's token under its client ID.
func (c *Controller) handleSet(req *agentapi.Request) *agentapi.Response {
	st := c.tokenStore()
	if st == nil {
		return &agentapi.Response{Error: agentapi.RespLocked}
	}
	if err := st.Set(req.ClientID, req.Token); err != nil {
		if errors.Is(err, errInvalidClientID) {
			return &agentapi.Response{Error: errMsgInvalidClientID}
		}
		return &agentapi.Response{Error: errMsgSet}
	}
	return &agentapi.Response{OK: true}
}

// handleStatus reports whether the agent is locked, how many tokens are cached
// (when unlocked), and whether an agent key already exists on disk.
func (c *Controller) handleStatus() *agentapi.Response {
	st := c.tokenStore()
	resp := &agentapi.Response{OK: true, Locked: st == nil, Initialized: c.keyExists()}
	if st != nil {
		resp.Count = st.Len()
	}
	return resp
}

// handleUnlock loads (or creates) the data key from the request passphrase and
// switches the agent to an unlocked, disk-backed store. It is idempotent: unlocking
// an already-unlocked agent succeeds without re-reading the key.
func (c *Controller) handleUnlock(req *agentapi.Request) *agentapi.Response {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.store != nil {
		return &agentapi.Response{OK: true}
	}
	dataKey, created, err := loadOrCreateDataKey(c.keyFile, []byte(req.Passphrase))
	if err != nil {
		if errors.Is(err, errIncorrectPassphrase) {
			return &agentapi.Response{Error: errIncorrectPassphrase.Error()}
		}
		return &agentapi.Response{Error: errMsgUnlock}
	}
	c.store = newDiskStore(dataKey, c.tokenDir)
	if c.logger != nil {
		if created {
			c.logger.Info("generated a new agent key", "path", c.keyFile)
			// A new key can't decrypt token files written under a previous key
			// (e.g. when the key file was deleted while the tokens remained), so
			// warn that those cached tokens are orphaned and will be re-minted.
			if n := c.store.Len(); n > 0 {
				c.logger.Warn("found cached token files that predate the new agent key; they can't be decrypted and will be re-minted on the next get", "path", c.tokenDir, "count", n)
			}
		}
		c.logger.Info("agent unlocked")
	}
	return &agentapi.Response{OK: true}
}

// tokenStore returns the current token store, or nil when the agent is locked.
func (c *Controller) tokenStore() *store {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.store
}

// keyExists reports whether an agent key file already exists on disk.
func (c *Controller) keyExists() bool {
	if c.keyFile == "" {
		return false
	}
	_, err := os.Stat(c.keyFile)
	return err == nil
}
