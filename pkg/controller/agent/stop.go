package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// Stop connects to a running agent over its Unix domain socket and asks it to
// shut down by sending a STOP command. It returns an error when no agent is
// listening or the agent reports a failure.
func (c *Controller) Stop(ctx context.Context, logger *slog.Logger) error {
	path, err := agentapi.SocketPath(os.Getenv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}

	resp, err := agentapi.Send(ctx, path, &agentapi.Request{Command: agentapi.CommandStop})
	if err != nil {
		return err //nolint:wrapcheck // Send returns a descriptive error (e.g. ErrAgentNotRunning)
	}
	if !resp.OK {
		return fmt.Errorf("the agent failed to stop: %s", resp.Error)
	}

	logger.Info("ghtkn agent stopped")
	return nil
}
