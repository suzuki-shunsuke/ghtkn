package status

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// Run reports whether a ghtkn agent is running, whether it is locked, and how
// many access tokens it currently caches when unlocked. A stopped agent is a normal
// result, not an error, so this method returns nil in that case.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger) error {
	path, err := agentapi.SocketPath(os.Getenv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}

	resp, running, err := queryStatus(ctx, path)
	if err != nil {
		return err
	}
	switch {
	case !running:
		logger.Info("ghtkn agent is not running")
	case resp.Locked:
		logger.Info("ghtkn agent is running but locked", "socket", path)
	default:
		logger.Info("ghtkn agent is running and unlocked", "cached_tokens", resp.Count, "socket", path)
	}
	return nil
}

// queryStatus asks the agent at path for its status. The bool result is false (with
// a nil error and a nil response) when no agent is listening.
func queryStatus(ctx context.Context, path string) (*agentapi.Response, bool, error) {
	resp, err := agentapi.Send(ctx, path, &agentapi.Request{Command: agentapi.CommandStatus})
	if err != nil {
		if agentapi.IsNotRunning(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("query the agent status: %w", err)
	}
	if !resp.OK {
		return nil, false, fmt.Errorf("the agent returned an error: %s", resp.Error)
	}
	return resp, true, nil
}
