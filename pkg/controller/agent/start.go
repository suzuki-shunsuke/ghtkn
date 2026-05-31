package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// Start runs the agent server in the foreground.
// It resolves and opens the Unix domain socket, serves clients until ctx is
// canceled, then removes the socket and exits.
//
// ctx is canceled when the process receives SIGINT or SIGTERM; the signal
// handling is set up by urfave.Main (see cmd/ghtkn/main.go), so this function
// does not register its own signal handler.
func (c *Controller) Start(ctx context.Context, logger *slog.Logger) error {
	path, err := socketPath()
	if err != nil {
		return err
	}

	listener, err := listen(ctx, path)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	defer listener.Close()

	logger.Info("ghtkn agent started", "socket", path)

	// Close the listener when the context is canceled so that serve returns.
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
