// Package get provides functionality to retrieve GitHub App access tokens.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package get

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/apptoken"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

// Controller manages the process of retrieving GitHub App access tokens.
// It coordinates between configuration reading, token caching, and token generation.
type Controller struct {
	input *Input
}

// New creates a new Controller instance with the provided input configuration.
func New(input *Input) *Controller {
	return &Controller{
		input: input,
	}
}

// Input contains all the dependencies and configuration needed by the Controller.
// It encapsulates file system access, configuration reading, token generation, and output handling.
type Input struct {
	ConfigFilePath string           // Path to the configuration file
	OutputFormat   string           // Output format ("json" or empty for plain text)
	MinExpiration  time.Duration    // Minimum token expiration duration required
	FS             afero.Fs         // File system abstraction for testing
	ConfigReader   ConfigReader     // Configuration file reader
	Env            *config.Env      // Environment variable provider
	AppTokenClient AppTokenClient   // Client for creating GitHub App tokens
	Stdout         io.Writer        // Output writer
	Keyring        Keyring          // Keyring for token storage
	Now            func() time.Time // Current time provider for testing
}

// NewInput creates a new Input instance with default production values.
// It sets up all necessary dependencies including file system, HTTP client, and keyring access.
func NewInput(configFilePath string) *Input {
	fs := afero.NewOsFs()
	return &Input{
		ConfigFilePath: configFilePath,
		FS:             fs,
		ConfigReader:   config.NewReader(fs),
		Env:            config.NewEnv(os.Getenv, runtime.GOOS),
		AppTokenClient: apptoken.NewClient(apptoken.NewInput()),
		Stdout:         os.Stdout,
		Keyring:        keyring.New(keyring.NewInput()),
		Now:            time.Now,
	}
}

// IsJSON returns true if the output format is set to JSON.
func (i *Input) IsJSON() bool {
	return i.OutputFormat == "json"
}

// Validate checks if the Input configuration is valid.
// It returns an error if the output format is neither empty nor "json".
func (i *Input) Validate() error {
	if i.OutputFormat != "" && !i.IsJSON() {
		return errors.New("output format must be empty or 'json'")
	}
	return nil
}

// ConfigReader defines the interface for reading configuration files.
type ConfigReader interface {
	Read(cfg *config.Config, configFilePath string) error
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
