package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// errAgentRunning is returned when another agent is already listening on the socket.
var errAgentRunning = errors.New("another ghtkn agent is already running")

// File system permissions for the agent socket and its parent directory.
const (
	socketDirPerm  os.FileMode = 0o700 // accessible only by the current user
	socketFilePerm os.FileMode = 0o600 // accessible only by the current user
)

// listen creates the Unix domain socket listener at path.
// It creates the parent directory, removes a stale socket left by a crashed agent,
// refuses to start when a live agent is already listening, and restricts the socket
// to the current user.
func listen(ctx context.Context, path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), socketDirPerm); err != nil {
		return nil, fmt.Errorf("create the socket directory: %w", err)
	}
	if err := cleanupStaleSocket(ctx, path); err != nil {
		return nil, err
	}
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on the socket: %w", err)
	}
	if err := os.Chmod(path, socketFilePerm); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("restrict the socket permissions: %w", err)
	}
	return listener, nil
}

// cleanupStaleSocket inspects an existing socket file at path.
// If a live agent answers on it, it returns errAgentRunning. Otherwise the file is
// treated as stale and removed so a new listener can be created.
func cleanupStaleSocket(ctx context.Context, path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat the socket: %w", err)
	}
	dialer := &net.Dialer{Timeout: agentapi.DialTimeout}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err == nil {
		conn.Close()
		return errAgentRunning
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove the stale socket: %w", err)
	}
	return nil
}
