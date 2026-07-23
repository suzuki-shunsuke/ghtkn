package lock

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// Run connects to a running agent over its Unix domain socket and asks it to discard its
// in-memory data key by sending a LOCK command, returning it to the locked state without
// stopping it. Locking an agent that is not running is a normal result (there is nothing
// unlocked to protect), so it returns nil in that case. It returns an error when the
// agent is reachable but reports a failure, or on an unexpected protocol error; an agent
// too old to know the LOCK command reports "unknown command", which surfaces here.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger) error {
	path, err := agentapi.SocketPath(c.getEnv, runtime.GOOS)
	if err != nil {
		return err //nolint:wrapcheck
	}

	resp, err := agentapi.Send(ctx, path, &agentapi.Request{Command: agentapi.CommandLock})
	if err != nil {
		if agentapi.IsNotRunning(err) {
			logger.Info("ghtkn agent is not running")
			return nil
		}
		return err //nolint:wrapcheck
	}
	if !resp.OK {
		return fmt.Errorf("the agent failed to lock (an agent from an older ghtkn may not support it; restart it with `ghtkn agent stop` then `ghtkn agent start`): %s", resp.Error)
	}

	logger.Info("ghtkn agent locked; run `ghtkn agent unlock` to use it again")
	return nil
}
