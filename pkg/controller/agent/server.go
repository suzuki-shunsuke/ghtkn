package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
)

// Error messages returned to clients in Response.Error.
const (
	errMsgEmptyRequest   = "empty request"
	errMsgInvalidRequest = "invalid request"
	errMsgReadRequest    = "read the request"
	errMsgUnknownCommand = "unknown command"
	errMsgNotFound       = "not found"
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
func (c *Controller) handleConn(conn net.Conn, logger *slog.Logger) {
	defer conn.Close()
	resp := c.handle(conn)
	b, err := json.Marshal(resp)
	if err != nil {
		logger.Error("marshal the agent response", "error", err)
		return
	}
	if _, err := conn.Write(append(b, '\n')); err != nil {
		logger.Error("write the agent response", "error", err)
	}
}

// handle reads and processes one request, returning the response to send.
func (c *Controller) handle(r io.Reader) *Response {
	line, err := bufio.NewReader(r).ReadBytes('\n')
	// ReadBytes returns io.EOF together with the data when there is no trailing
	// newline, so a non-empty line is still valid in that case.
	if err != nil && !errors.Is(err, io.EOF) {
		return &Response{Error: errMsgReadRequest}
	}
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return &Response{Error: errMsgEmptyRequest}
	}
	req := &Request{}
	if err := json.Unmarshal(line, req); err != nil {
		return &Response{Error: errMsgInvalidRequest}
	}
	return c.dispatch(req)
}

// dispatch routes a request to the matching handler.
func (c *Controller) dispatch(req *Request) *Response {
	switch req.Command {
	case CommandGet:
		token, ok := c.store.Get(req.ClientID)
		if !ok {
			return &Response{Error: errMsgNotFound}
		}
		return &Response{OK: true, Token: token}
	case CommandSet:
		c.store.Set(req.ClientID, req.Token)
		return &Response{OK: true}
	case CommandStatus:
		return &Response{OK: true, Count: c.store.Len()}
	default:
		return &Response{Error: errMsgUnknownCommand}
	}
}
