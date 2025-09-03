package get

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn/pkg/api"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// Run executes the main logic for retrieving a GitHub App access token.
// It reads configuration, checks for cached tokens, creates new tokens if needed,
// retrieves the authenticated user's login for Git Credential Helper if necessary,
// and outputs the result in the requested format.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger) error {
	cfg := &config.Config{}
	if err := c.readConfig(cfg); err != nil {
		return err
	}

	// Select the app config
	app := cfg.SelectApp(c.input.Env.App)
	logFields := []any{"app", app.Name}
	logger = logger.With(logFields...)

	token, err := c.input.TokenManager.Get(ctx, logger, &api.InputGet{
		ClientID:   app.ClientID,
		UseKeyring: cfg.Persist,
	})
	if err != nil {
		return fmt.Errorf("get access token: %w", slogerr.With(err, logFields...))
	}

	// Output access token
	if err := c.output(token); err != nil {
		return err
	}

	return nil
}

// readConfig loads and validates the configuration from the configured file path.
// It returns an error if the configuration cannot be read or is invalid.
func (c *Controller) readConfig(cfg *config.Config) error {
	if err := c.input.ConfigReader.Read(cfg, c.input.ConfigFilePath); err != nil {
		return fmt.Errorf("read config: %w", slogerr.With(err, "config", c.input.ConfigFilePath))
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
}
