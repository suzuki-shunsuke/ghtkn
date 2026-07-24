package unlock

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/harden"
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
	// Best-effort, before the passphrase is read: block same-user memory reads and core
	// dumps of this process (Linux-only, no-op elsewhere). This command is usually
	// short-lived, but it holds the passphrase while it waits at the refresh-token
	// removal prompt, which is bounded only by the user, and the marshaled request
	// carries a copy that cannot be zeroed (see agent.SecretBytes.MarshalJSON).
	harden.Process(logger)

	path, err := agentapi.SocketPath(c.getEnv, runtime.GOOS)
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
	// Ctrl-C) if the refresh setting is not what they meant. When refresh is off, the
	// agent may additionally prompt to confirm dropping stored refresh tokens after the
	// passphrase is entered (see doUnlock).
	logRefreshIntent(logger, enableRefreshToken)

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

	resp, err := c.doUnlock(ctx, logger, path, pass, enableRefreshToken, refreshTokenTTL)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil // the user declined dropping the refresh tokens; the agent stays locked.
	}
	if !resp.OK {
		return fmt.Errorf("unlock the agent: %s", resp.Error)
	}

	logger.Info("ghtkn agent unlocked", "refresh_token_enabled", resp.RefreshTokenEnabled)
	return nil
}

// logRefreshIntent surfaces, before the passphrase is entered, whether this unlock will
// enable or disable refresh tokens so the user can abort a mistaken setting.
func logRefreshIntent(logger *slog.Logger, enableRefreshToken bool) {
	if enableRefreshToken {
		logger.Info("refresh tokens will be enabled for this agent")
	} else {
		logger.Info("refresh tokens will be disabled for this agent")
	}
}

// doUnlock sends the unlock request and, when the agent reports RefreshTokenRemovalPending
// (unlocking without --enable-refresh while a still-valid refresh token is stored), prompts
// the user and re-sends with the confirmation set. It returns a nil response (and nil error)
// when the user declines, so the caller aborts and the agent stays locked. pass is passed
// directly (not as a string) so Run's deferred scrub zeroes the copy the request carries.
func (c *Controller) doUnlock(ctx context.Context, logger *slog.Logger, path string, pass []byte, enableRefreshToken bool, refreshTokenTTL time.Duration) (*agentapi.Response, error) {
	resp, err := agentapi.Send(ctx, path, &agentapi.Request{
		Command:            agentapi.CommandUnlock,
		Passphrase:         pass,
		EnableRefreshToken: enableRefreshToken,
		RefreshTokenTTL:    refreshTokenTTL,
	})
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	if resp.RefreshTokenRemovalPending {
		return c.confirmRefreshRemoval(ctx, logger, path, pass, enableRefreshToken, refreshTokenTTL)
	}
	return resp, nil
}

// confirmRefreshRemoval prompts the user (default No) to confirm dropping the stored
// refresh tokens and, on yes, re-sends the unlock with ConfirmRefreshTokenRemoval set,
// returning the agent's response. On no it logs the abort and returns (nil, nil) so the
// caller stops without unlocking; the agent stays locked. pass is reused as is: its scrub
// runs only when Run returns.
func (c *Controller) confirmRefreshRemoval(ctx context.Context, logger *slog.Logger, path string, pass []byte, enableRefreshToken bool, refreshTokenTTL time.Duration) (*agentapi.Response, error) {
	ok, err := c.confirm("Stored refresh tokens will be dropped (access tokens are kept; affected apps re-authenticate on next expiry). Rerun with --enable-refresh to keep them. Continue? (y/N): ")
	if err != nil {
		return nil, fmt.Errorf("confirm dropping stored refresh tokens: %w", err)
	}
	if !ok {
		logger.Info("unlock aborted; rerun with --enable-refresh to keep the stored refresh tokens")
		return nil, nil //nolint:nilnil // (nil, nil) signals a user-declined abort, distinct from an error.
	}
	resp, err := agentapi.Send(ctx, path, &agentapi.Request{
		Command:                    agentapi.CommandUnlock,
		Passphrase:                 pass,
		EnableRefreshToken:         enableRefreshToken,
		RefreshTokenTTL:            refreshTokenTTL,
		ConfirmRefreshTokenRemoval: true,
	})
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return resp, nil
}
