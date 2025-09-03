package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

type Logger struct {
	Expire func(logger *slog.Logger, exDate string)
}

func NewLogger() *Logger {
	return &Logger{
		Expire: func(logger *slog.Logger, exDate string) {
			logger.Debug("access token expires", "expiration_date", exDate)
		},
	}
}

type InputGet struct {
	ClientID   string
	UseKeyring bool
}

// Get executes the main logic for retrieving a GitHub App access token.
// It checks for cached tokens, creates new tokens if needed,
// retrieves the authenticated user's login for Git Credential Helper if necessary,
// and outputs the result in the requested format.
func (tm *TokenManager) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*keyring.AccessToken, error) {
	token, changed, err := tm.getOrCreateToken(ctx, logger, input)
	if err != nil {
		return nil, fmt.Errorf("get or create token: %w", err)
	}

	if token.Login == "" {
		// Get the authenticated user info for Git Credential Helper.
		// Git Credential Helper requires both username and password for authentication.
		// The username is the GitHub user's login name retrieved via the GitHub API.
		gh := tm.input.NewGitHub(ctx, token.AccessToken)
		user, err := gh.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("get authenticated user: %w", err)
		}
		token.Login = user.Login
		changed = true
	}

	if input.UseKeyring && changed {
		// Store the token in keyring
		if err := tm.input.Keyring.Set(input.ClientID, &keyring.AccessToken{
			AccessToken:    token.AccessToken,
			ExpirationDate: token.ExpirationDate,
			Login:          token.Login,
		}); err != nil {
			return token, ErrStoreToken
		}
	}

	return token, nil
}

var ErrStoreToken = errors.New("could not store the token in keyring")

// getOrCreateToken retrieves an existing token from the keyring or creates a new one.
// It returns the token, a boolean indicating whether the token was newly created or modified,
// and any error that occurred. The changed flag is used to determine if the token should be
// saved back to the keyring.
func (tm *TokenManager) getOrCreateToken(ctx context.Context, logger *slog.Logger, input *InputGet) (*keyring.AccessToken, bool, error) {
	// Get an access token from keyring
	if input.UseKeyring {
		token, err := tm.getAccessTokenFromKeyring(logger, input.ClientID)
		if err != nil {
			slogerr.WithError(logger, err).Info("failed to get a GitHub App User Access Token from keyring")
		}
		if token != nil {
			return token, false, nil
		}
	}
	// Create access token
	token, err := tm.createToken(ctx, logger, input.ClientID)
	if err != nil {
		return nil, false, fmt.Errorf("create a GitHub App User Access Token: %w", err)
	}
	return token, true, nil
}

// createToken generates a new GitHub App access token using the OAuth device flow.
// It returns a keyring.AccessToken with the token details and expiration date.
func (tm *TokenManager) createToken(ctx context.Context, logger *slog.Logger, clientID string) (*keyring.AccessToken, error) {
	tk, err := tm.input.AppTokenClient.Create(ctx, logger, clientID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return &keyring.AccessToken{
		AccessToken:    tk.AccessToken,
		ExpirationDate: tk.ExpirationDate,
	}, nil
}

// getAccessTokenFromKeyring retrieves a cached access token from the system keyring.
// It returns nil if the token doesn't exist or has expired based on MinExpiration.
func (tm *TokenManager) getAccessTokenFromKeyring(logger *slog.Logger, clientID string) (*keyring.AccessToken, error) {
	// Get an access token from keyring
	tk, err := tm.input.Keyring.Get(clientID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	if tk == nil {
		return nil, nil //nolint:nilnil
	}
	// Check if the access token expires
	expired, err := tm.checkExpired(tk.ExpirationDate)
	if err != nil {
		return nil, fmt.Errorf("check if the access token is expired: %w", err)
	}
	if expired {
		tm.input.Logger.Expire(logger, tk.ExpirationDate)
		return nil, nil //nolint:nilnil
	}
	// Not expires
	return tk, nil
}

// checkExpired determines if an access token should be considered expired.
// It returns true if the token will expire within the MinExpiration duration from now.
// This ensures tokens are renewed before they actually expire.
func (tm *TokenManager) checkExpired(exDate string) (bool, error) {
	t, err := keyring.ParseDate(exDate)
	if err != nil {
		return false, err //nolint:wrapcheck
	}
	// Expiration Date - Now < Min Expiration
	// Now + Min Expiration > Expiration Date
	return tm.input.Now().Add(tm.input.MinExpiration).After(t), nil
}
