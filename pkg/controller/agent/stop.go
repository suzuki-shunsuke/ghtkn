package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
)

// Stop connects to a running agent over its Unix domain socket and asks it to
// shut down by sending a STOP command. It returns an error when no agent is
// listening or the agent reports a failure.
func (c *Controller) Stop(ctx context.Context, logger *slog.Logger) error {
	path, err := socketPath()
	if err != nil {
		return err
	}

	dialer := &net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return fmt.Errorf("connect to the ghtkn agent (is it running?): %w", err)
	}
	defer conn.Close()

	req, err := json.Marshal(&Request{Command: CommandStop})
	if err != nil {
		return fmt.Errorf("marshal the stop request: %w", err)
	}
	if _, err := conn.Write(append(req, '\n')); err != nil {
		return fmt.Errorf("send the stop request: %w", err)
	}

	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read the stop response: %w", err)
	}
	resp := &Response{}
	if err := json.Unmarshal(line, resp); err != nil {
		return fmt.Errorf("parse the stop response: %w", err)
	}
	if !resp.OK {
		return fmt.Errorf("the agent failed to stop: %s", resp.Error)
	}

	logger.Info("ghtkn agent stopped")
	return nil
}
