package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Stop connects to a running agent over its Unix domain socket and asks it to
// shut down by sending a STOP command. It returns an error when no agent is
// listening or the agent reports a failure.
func (c *Controller) Stop(ctx context.Context, logger *slog.Logger) error {
	path, err := socketPath()
	if err != nil {
		return err
	}

	resp, err := request(ctx, path, &Request{Command: CommandStop})
	if err != nil {
		if isNotRunning(err) {
			return errors.New("the ghtkn agent is not running")
		}
		return fmt.Errorf("send the stop request: %w", err)
	}
	if !resp.OK {
		return fmt.Errorf("the agent failed to stop: %s", resp.Error)
	}

	logger.Info("ghtkn agent stopped")
	return nil
}
