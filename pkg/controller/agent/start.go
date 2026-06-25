package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/keystore"
)

// Start runs the agent server in the foreground.
// It opens the Unix domain socket and serves clients until ctx is canceled or a STOP
// command is received, then removes the socket and exits. The agent starts locked;
// clients use 'ghtkn agent unlock' to load the data key. Because Start needs no
// terminal, it can run as a background service.
//
// ctx is canceled when the process receives SIGINT or SIGTERM; the signal
// handling is set up by urfave.Main (see cmd/ghtkn/main.go), so this function
// does not register its own signal handler.
func (c *Controller) Start(ctx context.Context, logger *slog.Logger) error {
	keyFile, err := keystore.KeyPath(os.Getenv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}
	dir, err := keystore.TokenDir(os.Getenv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}
	c.keyFile = keyFile
	c.tokenDir = dir
	c.logger = logger

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.shutdown = cancel

	path, err := agentapi.SocketPath(os.Getenv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}

	listener, err := listen(ctx, path)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	defer listener.Close()

	logger.Info("ghtkn agent started", "socket", path, "locked", true)

	// Close the listener when the context is canceled (signal or STOP command)
	// so that serve returns.
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	if err := c.serve(listener, logger); err != nil {
		return fmt.Errorf("serve the agent socket: %w", err)
	}

	logger.Info("ghtkn agent stopped")
	return nil
}
