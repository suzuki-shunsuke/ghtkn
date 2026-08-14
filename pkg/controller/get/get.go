package get

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
)

type InputRun struct {
	InputGet *ghtkn.InputGet
}

// Run executes the main logic for retrieving a GitHub App access token.
// It reads configuration and the token cached in the backend, retrieves the
// authenticated user's login for Git Credential Helper if necessary, and outputs the
// result in the requested format. It fails when no valid token is cached, because
// creating one runs the device flow and only 'ghtkn auth' does that.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger, input *InputRun) error {
	token, app, err := c.input.Client.Get(ctx, logger, input.InputGet)
	if err != nil {
		return fmt.Errorf("get or create access token: %w", err)
	}

	// app is nil when the token comes from GHTKN_GITHUB_TOKEN; fall back to an
	// empty app name in that case.
	appName := ""
	if app != nil {
		appName = app.Name
	}
	// Output access token
	if err := c.output(appName, token); err != nil {
		return err
	}

	return nil
}
