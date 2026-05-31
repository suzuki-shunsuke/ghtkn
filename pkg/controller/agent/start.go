package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
)

// Start runs the agent server in the foreground.
// It prompts for the passphrase, loads (or creates) the encryption key, opens the
// Unix domain socket, and serves clients until ctx is canceled or a STOP command is
// received, then removes the socket and exits.
//
// ctx is canceled when the process receives SIGINT or SIGTERM; the signal
// handling is set up by urfave.Main (see cmd/ghtkn/main.go), so this function
// does not register its own signal handler.
func (c *Controller) Start(ctx context.Context, logger *slog.Logger) error {
	if err := c.unlock(logger); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.shutdown = cancel

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

// unlock prompts for the passphrase, loads or creates the data key, and switches
// the controller to a disk-backed encrypted store. It must be called before listen
// so that a wrong passphrase aborts startup without creating a socket.
func (c *Controller) unlock(logger *slog.Logger) error {
	keyFile, err := keyPath(os.Getenv, runtime.GOOS)
	if err != nil {
		return err
	}
	dir, err := tokenDir(os.Getenv, runtime.GOOS)
	if err != nil {
		return err
	}

	_, statErr := os.Stat(keyFile)
	exists := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("stat the key file: %w", statErr)
	}

	pass, err := c.promptPassphrase(exists)
	if err != nil {
		return err
	}
	// Best-effort scrubbing of the passphrase bytes. The garbage collector may
	// have copied them, so this is hygiene rather than a guarantee.
	defer func() {
		for i := range pass {
			pass[i] = 0
		}
	}()

	dataKey, created, err := loadOrCreateDataKey(keyFile, pass)
	if err != nil {
		return err
	}
	if created {
		logger.Info("generated a new agent key", "path", keyFile)
	}

	c.store = newDiskStore(dataKey, dir)
	logger.Info("agent unlocked")
	return nil
}
