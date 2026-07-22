package reset

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/harden"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/keyfile"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/stop"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tty"
)

// Run recovers from a forgotten passphrase by reinitializing the agent: it stops
// a running agent, deletes the key file and all encrypted token files, and creates a
// new key from a freshly entered passphrase. The old passphrase is not needed and
// the cached tokens are discarded (they are reminted from GitHub on the next get).
//
// It asks for confirmation first because the operation is destructive, and requires
// a terminal both for that confirmation and for the new passphrase.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger) error {
	// Best-effort, before the new passphrase is read: block same-user memory reads and
	// core dumps of this process (Linux-only, no-op elsewhere). It holds the passphrase
	// from the prompt until the new key file is written.
	harden.Process(logger)

	keyFile, err := keyfile.KeyPath(c.getEnv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}
	dir, err := tokenstore.TokenDir(c.getEnv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}

	ok, err := c.confirm("This stops the agent and deletes the key and all cached tokens, then recreates the key. Continue? (y/N): ")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("ghtkn agent reset was canceled")
		return nil
	}

	// Stop a running agent first so it does not keep using the old data key or
	// write tokens after the files are deleted. Stop is a no-op (nil) when no agent
	// is running.
	if err := stop.NewWithEnv(c.getEnv).Run(ctx, logger); err != nil {
		return err //nolint:wrapcheck
	}
	if err := deleteAgentFiles(keyFile, dir); err != nil {
		return err
	}
	if err := c.recreateKey(keyFile); err != nil {
		return err
	}

	logger.Info("ghtkn agent has been reset", "key", keyFile)
	return nil
}

// deleteAgentFiles removes the encrypted token directory and the key file. Tokens
// are deleted first because they are useless without the key.
func deleteAgentFiles(keyFile, tokenDir string) error {
	if err := os.RemoveAll(tokenDir); err != nil {
		return fmt.Errorf("delete the token directory: %w", err)
	}
	if err := os.Remove(keyFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete the key file: %w", err)
	}
	return nil
}

// recreateKey prompts for a new passphrase (twice, to confirm) and writes a new key
// file. The key file must not exist when this is called.
func (c *Controller) recreateKey(keyFile string) error {
	pass, err := tty.PromptPassphrase(c.readPassphrase, false)
	if err != nil {
		return err //nolint:wrapcheck
	}
	defer func() {
		for i := range pass {
			pass[i] = 0
		}
	}()
	if _, err := keyfile.CreateDataKey(keyFile, pass); err != nil {
		return err //nolint:wrapcheck
	}
	return nil
}
