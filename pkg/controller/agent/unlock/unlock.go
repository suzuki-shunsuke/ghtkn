package unlock

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tty"
)

// Run prompts for the agent passphrase on the terminal and sends it to a running
// agent over the socket, loading (or creating) the data key. It is the client half
// of the locked-start workflow: 'ghtkn agent start' runs locked in the background,
// and 'ghtkn agent unlock' supplies the passphrase interactively.
//
// enableRefreshToken binds refresh-token enablement to this passphrase-authenticated
// unlock. The current refresh state is logged so the user can notice if it was enabled
// without their intent (e.g. by an injected flag). refreshTokenTTL is how long a stored
// token may sit unused before the agent discards it; it applies only when refresh is
// enabled.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger, enableRefreshToken bool, refreshTokenTTL time.Duration) error {
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
		logger.Info("ghtkn agent is already unlocked", "refresh_token_enabled", status.RefreshTokenEnabled)
		return nil
	}

	// Surface the intent before the passphrase is entered, so the user can abort (e.g.
	// Ctrl-C) if they did not mean to enable refresh tokens.
	if enableRefreshToken {
		logger.Info("refresh tokens will be enabled for this agent")
	}

	// status.Initialized reports whether a key file already exists. On first use
	// (not initialized) PromptPassphrase asks twice and verifies the entries match.
	pass, err := tty.PromptPassphrase(c.readPassphrase, status.Initialized)
	if err != nil {
		return err //nolint:wrapcheck
	}
	// Best-effort scrubbing of the passphrase bytes.
	defer func() {
		for i := range pass {
			pass[i] = 0
		}
	}()

	resp, err := agentapi.Send(ctx, path, &agentapi.Request{
		Command:            agentapi.CommandUnlock,
		Passphrase:         string(pass),
		EnableRefreshToken: enableRefreshToken,
		RefreshTokenTTL:    refreshTokenTTL,
	})
	if err != nil {
		return err //nolint:wrapcheck
	}
	if !resp.OK {
		return fmt.Errorf("unlock the agent: %s", resp.Error)
	}

	logger.Info("ghtkn agent unlocked", "refresh_token_enabled", resp.RefreshTokenEnabled)
	return nil
}
