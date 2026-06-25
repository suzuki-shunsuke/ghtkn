package stop

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// Run connects to a running agent over its Unix domain socket and asks it to
// shut down by sending a STOP command. Stopping an agent that is not running is a
// normal result (like 'systemctl stop'), so it returns nil in that case. It returns
// an error only when the agent is reachable but reports a failure, or on an
// unexpected protocol error.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger) error {
	path, err := agentapi.SocketPath(os.Getenv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}

	resp, err := agentapi.Send(ctx, path, &agentapi.Request{Command: agentapi.CommandStop})
	if err != nil {
		if agentapi.IsNotRunning(err) {
			logger.Info("ghtkn agent is not running")
			return nil
		}
		return err //nolint:wrapcheck
	}
	if !resp.OK {
		return fmt.Errorf("the agent failed to stop: %s", resp.Error)
	}

	logger.Info("ghtkn agent stopped")
	return nil
}
