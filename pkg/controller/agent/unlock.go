package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// Unlock prompts for the agent passphrase on the terminal and sends it to a running
// agent over the socket, loading (or creating) the data key. It is the client half
// of the locked-start workflow: 'ghtkn agent start' runs locked in the background,
// and 'ghtkn agent unlock' supplies the passphrase interactively.
func (c *Controller) Unlock(ctx context.Context, logger *slog.Logger) error {
	path, err := agentapi.SocketPath(os.Getenv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}

	status, err := agentapi.Send(ctx, path, &agentapi.Request{Command: agentapi.CommandStatus})
	if err != nil {
		return err //nolint:wrapcheck // Send returns a descriptive error (e.g. ErrAgentNotRunning)
	}
	if !status.OK {
		return fmt.Errorf("query the agent status: %s", status.Error)
	}
	if !status.Locked {
		logger.Info("ghtkn agent is already unlocked")
		return nil
	}

	// status.Initialized reports whether a key file already exists. On first use
	// (not initialized) promptPassphrase asks twice and verifies the entries match.
	pass, err := c.promptPassphrase(status.Initialized)
	if err != nil {
		return err
	}
	// Best-effort scrubbing of the passphrase bytes.
	defer func() {
		for i := range pass {
			pass[i] = 0
		}
	}()

	resp, err := agentapi.Send(ctx, path, &agentapi.Request{
		Command:    agentapi.CommandUnlock,
		Passphrase: string(pass),
	})
	if err != nil {
		return err //nolint:wrapcheck
	}
	if !resp.OK {
		return fmt.Errorf("unlock the agent: %s", resp.Error)
	}

	logger.Info("ghtkn agent unlocked")
	return nil
}
