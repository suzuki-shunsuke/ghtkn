package get

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// Run executes the main logic for retrieving a GitHub App access token.
// It reads configuration, checks for cached tokens, creates new tokens if needed,
// and outputs the result in the requested format.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger) error {
	cfg := &config.Config{}
	if err := c.readConfig(cfg); err != nil {
		return err
	}

	// Select the app config
	app := cfg.SelectApp(c.input.Env.App)
	logFields := []any{"app", app.ID}
	logger = logger.With(logFields...)

	var token *keyring.AccessToken
	if cfg.Persist {
		// Get an access token from keyring
		tk, err := c.getAccessTokenFromKeyring(logger, app)
		if err != nil {
			logger.Warn("get a GitHub App User Access Token from keyring", "error", err)
		}
		token = tk
	}
	if token == nil {
		// Create access token
		tk, err := c.createToken(ctx, logger, app)
		if err != nil {
			return fmt.Errorf("create a GitHub App User Access Token: %w", slogerr.With(err, logFields...))
		}
		token = tk
	}

	// Output access token
	if err := c.output(token); err != nil {
		return err
	}

	if cfg.Persist {
		// Store the token in keyring
		if err := c.input.Keyring.Set(app.ClientID, &keyring.AccessToken{
			AccessToken:    token.AccessToken,
			ExpirationDate: token.ExpirationDate,
		}); err != nil {
			logger.Warn("could not store a GitHub App User Access Token in keyring", "error", err)
		}
	}

	return nil
}

// createToken generates a new GitHub App access token using the OAuth device flow.
// It returns a keyring.AccessToken with the token details and expiration date.
func (c *Controller) createToken(ctx context.Context, logger *slog.Logger, app *config.App) (*keyring.AccessToken, error) {
	tk, err := c.input.AppTokenClient.Create(ctx, logger, app.ClientID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return &keyring.AccessToken{
		App:            app.ID,
		AccessToken:    tk.AccessToken,
		ExpirationDate: tk.ExpirationDate,
	}, nil
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

// getAccessTokenFromKeyring retrieves a cached access token from the system keyring.
// It returns nil if the token doesn't exist or has expired based on MinExpiration.
func (c *Controller) getAccessTokenFromKeyring(logger *slog.Logger, app *config.App) (*keyring.AccessToken, error) {
	// Get an access token from keyring
	tk, err := c.input.Keyring.Get(app.ClientID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	if tk == nil {
		return nil, nil //nolint:nilnil
	}
	tk.App = app.ID
	// Check if the access token expires
	expired, err := c.checkExpired(tk.ExpirationDate)
	if err != nil {
		return nil, fmt.Errorf("check if the access token is expired: %w", err)
	}
	if expired {
		logger.Debug("access token expires", "expiration_date", tk.ExpirationDate)
		return nil, nil //nolint:nilnil
	}
	// Not expires
	return tk, nil
}

// checkExpired determines if an access token should be considered expired.
// It returns true if the token will expire within the MinExpiration duration from now.
// This ensures tokens are renewed before they actually expire.
func (c *Controller) checkExpired(exDate string) (bool, error) {
	t, err := keyring.ParseDate(exDate)
	if err != nil {
		return false, err //nolint:wrapcheck
	}
	// Expiration Date - Now < Min Expiration
	// Now + Min Expiration > Expiration Date
	return c.input.Now().Add(c.input.MinExpiration).After(t), nil
}
