package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// Status reports whether a ghtkn agent is running and, if so, how many access
// tokens it currently caches. A stopped agent is a normal result, not an error,
// so this method returns nil in that case.
func (c *Controller) Status(ctx context.Context, logger *slog.Logger) error {
	path, err := agentapi.SocketPath(os.Getenv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}

	running, count, err := queryStatus(ctx, path)
	if err != nil {
		return err
	}
	if !running {
		logger.Info("ghtkn agent is not running")
		return nil
	}
	logger.Info("ghtkn agent is running", "cached_tokens", count, "socket", path)
	return nil
}

// queryStatus asks the agent at path for its status. It returns running=false
// (with a nil error) when no agent is listening.
func queryStatus(ctx context.Context, path string) (bool, int, error) {
	resp, err := agentapi.Send(ctx, path, &agentapi.Request{Command: agentapi.CommandStatus})
	if err != nil {
		if agentapi.IsNotRunning(err) {
			return false, 0, nil
		}
		return false, 0, fmt.Errorf("query the agent status: %w", err)
	}
	if !resp.OK {
		return false, 0, fmt.Errorf("the agent returned an error: %s", resp.Error)
	}
	return true, resp.Count, nil
}
