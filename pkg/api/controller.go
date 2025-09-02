// Package api provides functionality to retrieve GitHub App access tokens.
// It serves both the standard 'get' command and the 'git-credential' helper command.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package api

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/apptoken"
	"github.com/suzuki-shunsuke/ghtkn/pkg/github"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

// TokenManager manages the process of retrieving GitHub App access tokens.
// It coordinates between configuration reading, token caching, and token generation.
type TokenManager struct {
	input *Input
}

// New creates a new Controller instance with the provided input configuration.
func New(input *Input) *TokenManager {
	return &TokenManager{
		input: input,
	}
}

// Input contains all the dependencies and configuration needed by the Controller.
// It encapsulates file system access, configuration reading, token generation, and output handling.
// The IsGitCredential flag determines whether to format output for Git's credential helper protocol.
type Input struct {
	OutputFormat   string           // Output format ("json" or empty for plain text)
	MinExpiration  time.Duration    // Minimum token expiration duration required
	FS             afero.Fs         // File system abstraction for testing
	AppTokenClient AppTokenClient   // Client for creating GitHub App tokens
	Stdout         io.Writer        // Output writer
	Keyring        Keyring          // Keyring for token storage
	Now            func() time.Time // Current time provider for testing
	NewGitHub      func(ctx context.Context, token string) GitHub
}

// NewInput creates a new Input instance with default production values.
// It sets up all necessary dependencies including file system, HTTP client, and keyring access.
func NewInput() *Input {
	fs := afero.NewOsFs()
	return &Input{
		FS:             fs,
		AppTokenClient: apptoken.NewClient(apptoken.NewInput()),
		Stdout:         os.Stdout,
		Keyring:        keyring.New(keyring.NewInput()),
		Now:            time.Now,
		NewGitHub: func(ctx context.Context, token string) GitHub {
			return github.New(ctx, token)
		},
	}
}

// Validate checks if the Input configuration is valid.
// It returns an error if the output format is neither empty nor "json".
func (i *Input) Validate() error {
	return nil
}

// AppTokenClient defines the interface for creating GitHub App access tokens.
type AppTokenClient interface {
	Create(ctx context.Context, logger *slog.Logger, clientID string) (*apptoken.AccessToken, error)
}

// Keyring defines the interface for storing and retrieving tokens from the system keyring.
type Keyring interface {
	Get(key string) (*keyring.AccessToken, error)
	Set(key string, token *keyring.AccessToken) error
}

// GitHub defines the interface for interacting with the GitHub API.
// It is used to retrieve authenticated user information needed for Git Credential Helper.
type GitHub interface {
	GetUser(ctx context.Context) (*github.User, error)
}
