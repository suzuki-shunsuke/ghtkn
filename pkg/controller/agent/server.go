package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
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
		return &agentapi.Response{OK: true, Count: c.store.Len()}, false
	case agentapi.CommandStop:
		return &agentapi.Response{OK: true}, true
	default:
		return &agentapi.Response{Error: errMsgUnknownCommand}, false
	}
}

// handleGet returns the cached token for the request's client ID.
func (c *Controller) handleGet(req *agentapi.Request) *agentapi.Response {
	token, ok, err := c.store.Get(req.ClientID)
	switch {
	case errors.Is(err, errInvalidClientID):
		return &agentapi.Response{Error: errMsgInvalidClientID}
	case err != nil:
		return &agentapi.Response{Error: errMsgGet}
	case !ok:
		return &agentapi.Response{Error: agentapi.RespNotFound}
	}
	return &agentapi.Response{OK: true, Token: token}
}

// handleSet stores the request's token under its client ID.
func (c *Controller) handleSet(req *agentapi.Request) *agentapi.Response {
	if err := c.store.Set(req.ClientID, req.Token); err != nil {
		if errors.Is(err, errInvalidClientID) {
			return &agentapi.Response{Error: errMsgInvalidClientID}
		}
		return &agentapi.Response{Error: errMsgSet}
	}
	return &agentapi.Response{OK: true}
}
