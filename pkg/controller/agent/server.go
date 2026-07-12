package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
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
	errMsgStartDeviceFlow = "start the device flow"
	errMsgDelete          = "delete the token"
	errMsgUnlock          = "unlock the agent"
)

// serve accepts connections until the listener is closed and handles each one.
// It returns nil when the listener is closed (e.g. on shutdown). ctx is the server's
// context; it is passed to handlers so a device flow started to satisfy a GET keeps
// running after the request connection closes and stops on server shutdown.
func (c *Controller) serve(ctx context.Context, listener net.Listener, logger *slog.Logger) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err //nolint:wrapcheck
		}
		go c.handleConn(ctx, conn, logger)
	}
}

// handleConn reads a single request from conn, processes it, writes the response,
// and closes the connection. Each connection serves exactly one request.
// When the request asks the agent to stop, the shutdown is triggered only after
// the response has been written so the client always receives the acknowledgment.
func (c *Controller) handleConn(ctx context.Context, conn net.Conn, logger *slog.Logger) {
	defer conn.Close()
	resp, shutdown := c.handle(ctx, conn)
	b, err := json.Marshal(resp)
	if err != nil {
		logger.Error("marshal the agent response", "error", err)
		return
	}
	b = append(b, '\n')
	if _, err := conn.Write(b); err != nil {
		logger.Error("write the agent response", "error", err)
	}
	// The marshaled response may carry an access token (GET). Zero it and the stored
	// token bytes once written so the plaintext does not linger in memory.
	scrub(b)
	scrub(resp.Token)
	if shutdown && c.shutdown != nil {
		c.shutdown()
	}
}

// handle reads and processes one request, returning the response to send and
// whether the agent should shut down afterwards.
func (c *Controller) handle(ctx context.Context, r io.Reader) (*agentapi.Response, bool) {
	line, err := bufio.NewReader(r).ReadBytes('\n')
	// An UNLOCK request line carries the passphrase; zero the request bytes once the
	// request has been dispatched so the plaintext does not linger. req.Passphrase is a
	// separate copy (see SecretBytes.UnmarshalJSON) and is scrubbed by handleUnlock.
	defer scrub(line)
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
	// The agent serves any version in [MinProtocolVersion, ProtocolVersion], handling
	// an older but still-supported client with that older version's behavior (see
	// dispatch) so old clients keep working after the agent is upgraded. A client
	// below the minimum is too old to serve, and one above this agent's version means
	// this agent itself is out of date; both fail fast with a clear "upgrade" message.
	switch {
	case req.ProtocolVersion < agentapi.MinProtocolVersion:
		return &agentapi.Response{Error: agentapi.RespObsoleteClient}, false
	case req.ProtocolVersion > agentapi.ProtocolVersion:
		return &agentapi.Response{Error: agentapi.RespObsoleteAgent}, false
	}
	return c.dispatch(ctx, req)
}

// dispatch routes a request to the matching handler.
// The second return value reports whether the agent should shut down.
//
// A legacy client (protocol version below ProtocolVersionServerLifecycle) predates
// the server taking over the token lifecycle: it mints tokens itself and stores them
// with SET, and the agent must not run the device flow or refresh for it. Such a
// client never sets StartDeviceFlow (the field did not exist), so GET never starts a
// flow on its own; refresh is disabled explicitly here.
func (c *Controller) dispatch(ctx context.Context, req *agentapi.Request) (*agentapi.Response, bool) {
	legacy := req.ProtocolVersion < agentapi.ProtocolVersionServerLifecycle
	switch req.Command {
	case agentapi.CommandGet:
		return c.handleGet(ctx, req, c.refreshEnabled() && !legacy), false
	case agentapi.CommandSet:
		return c.handleSet(req), false
	case agentapi.CommandRevoke:
		return c.handleRevoke(ctx, req), false
	case agentapi.CommandDelete:
		return c.handleDelete(req), false
	case agentapi.CommandStatus:
		return c.handleStatus(), false
	case agentapi.CommandUnlock:
		return c.handleUnlock(ctx, req), false
	case agentapi.CommandStop:
		return &agentapi.Response{OK: true}, true
	default:
		return &agentapi.Response{Error: errMsgUnknownCommand}, false
	}
}

// readStoredToken reads the stored token for clientID and classifies store errors. It
// returns the raw token and whether one is present, plus a non-nil response to send
// back on a hard error. An undecryptable token (e.g. after a key rotation) is reported
// as a miss (ok false, resp nil) so the caller re-mints it instead of failing with an
// opaque error.
func (c *Controller) readStoredToken(st *tokenstore.Store, clientID string) (json.RawMessage, bool, *agentapi.Response) {
	token, ok, err := st.Get(clientID)
	switch {
	case errors.Is(err, tokenstore.ErrInvalidClientID):
		return nil, false, &agentapi.Response{Error: errMsgInvalidClientID}
	case errors.Is(err, tokenstore.ErrDecryptToken):
		if c.logger != nil {
			slogerr.WithError(c.logger, err).Warn("discard an undecryptable cached token", "client_id", clientID)
		}
		return nil, false, nil
	case err != nil:
		return nil, false, &agentapi.Response{Error: fmt.Sprintf("%s: %s", errMsgGet, err)}
	}
	return token, ok, nil
}

// tokenStore returns the current token store, or nil when the agent is locked.
func (c *Controller) tokenStore() *tokenstore.Store {
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
