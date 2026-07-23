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
	"time"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// Bounds on a single connection so a stalled or oversized local request cannot park a
// handler goroutine (and its file descriptor) forever or force unbounded buffering. The
// socket is restricted to the current user, so these guard against a buggy client rather
// than a remote attacker.
const (
	// readRequestTimeout bounds how long a connection may take to send its one request
	// line. A client builds the whole request and writes it at once, so this is generous.
	readRequestTimeout = 10 * time.Second
	// writeResponseTimeout bounds how long writing the response may block, so a client
	// that stops reading cannot wedge the handler.
	writeResponseTimeout = 10 * time.Second
	// maxRequestBytes caps the request line so a connection that never sends a newline
	// cannot force the read buffer to grow without limit. It is far above any legitimate
	// request (the largest carries an UNLOCK passphrase or a client-minted token).
	maxRequestBytes = 64 * 1024
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
	errMsgSetNotAllowed   = "set is not allowed for this protocol version; the agent owns the token lifecycle"
	errMsgStartDeviceFlow = "start the device flow"
	// errMsgDeviceFlowFailed is returned to a poll waiting on a device flow that ended
	// without minting a token (the one-time code expired, or the poll/store failed). It
	// is a full sentence because the client prints it verbatim.
	errMsgDeviceFlowFailed = "the ghtkn agent's device flow did not complete; the one-time code may have expired. Run the command again to retry."
	errMsgDelete           = "delete the token"
	errMsgUnlock           = "unlock the agent"
	// errMsgRefreshTokenRemovalPending accompanies RefreshTokenRemovalPending so an older
	// client that does not understand the field still shows a meaningful reason.
	errMsgRefreshTokenRemovalPending = "stored refresh tokens would be removed; confirm the removal or rerun with --enable-refresh to keep them"
	// errMsgRefreshTokenUnsupportedOS is returned when an UNLOCK asks to enable refresh
	// tokens on an OS that doesn't support them (see RefreshTokenSupported).
	errMsgRefreshTokenUnsupportedOS = "refresh tokens are not supported on Windows; unlock the agent without enabling them"
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
	// Bound the read: a client that connects but never sends a full request line must not
	// park this goroutine forever, and it must not force the buffer to grow without limit.
	if err := conn.SetReadDeadline(time.Now().Add(readRequestTimeout)); err != nil {
		logger.Error("set the read deadline", "error", err)
		return
	}
	resp, shutdown := c.handle(ctx, io.LimitReader(conn, maxRequestBytes))
	// Stamp this agent's protocol version on every response so a client can tell how old
	// the agent is. A client that needs the server-owned token lifecycle refuses an agent
	// that does not set it (agentapi.ErrObsoleteAgent): such an agent predates the
	// request fields the lifecycle depends on and would answer them as a plain GET.
	// Upgrading ghtkn does not update a running agent, so this is the signal that the
	// user must restart it.
	resp.ProtocolVersion = agentapi.ProtocolVersion
	// The response may carry an access token (GET); zero it and the marshaled bytes once no
	// longer needed so the plaintext does not linger in memory. Deferred so every path below
	// scrubs, including an early return on a write-deadline error.
	defer scrub(resp.Token)
	b, err := json.Marshal(resp)
	if err != nil {
		logger.Error("marshal the agent response", "error", err)
		return
	}
	b = append(b, '\n')
	defer scrub(b)
	// Bound the write similarly, so a client that stops reading cannot wedge the handler.
	if err := conn.SetWriteDeadline(time.Now().Add(writeResponseTimeout)); err != nil {
		logger.Error("set the write deadline", "error", err)
		return
	}
	if _, err := conn.Write(b); err != nil {
		logger.Error("write the agent response", "error", err)
	}
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
	// A SET request (legacy client) carries a client-minted token; json.Unmarshal copied
	// it out of line into its own buffer, so scrub it once dispatch is done. handleSet's
	// Store.Set takes its own copy, so this does not corrupt the stored token.
	defer scrub(req.Token)
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
		return c.handleSet(req, legacy), false
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
