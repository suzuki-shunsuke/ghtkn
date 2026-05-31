package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
)

// request opens a connection to the agent at path, sends a single
// newline-delimited JSON request, and reads the single newline-delimited JSON
// response. The dial error is returned unwrapped so callers can classify it
// (e.g. treat a missing or dead socket as "the agent is not running").
func request(ctx context.Context, path string, req *Request) (*Response, error) {
	dialer := &net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return nil, err //nolint:wrapcheck // callers classify the dial error with isNotRunning
	}
	defer conn.Close()

	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal the request: %w", err)
	}
	if _, err := conn.Write(append(b, '\n')); err != nil {
		return nil, fmt.Errorf("send the request: %w", err)
	}

	// ReadBytes returns io.EOF together with the data when the server closes the
	// connection without a trailing newline, so a non-empty line is still valid.
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read the response: %w", err)
	}
	resp := &Response{}
	if err := json.Unmarshal(line, resp); err != nil {
		return nil, fmt.Errorf("parse the response: %w", err)
	}
	return resp, nil
}

// isNotRunning reports whether err from request indicates that no agent is
// listening: either the socket file is absent or nothing accepts connections on
// it (a stale socket left by a crashed agent).
func isNotRunning(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED)
}
