// Package get provides functionality to retrieve GitHub App access tokens.
// It serves both the standard 'get' command and the 'git-credential' helper command.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package get

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"runtime"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/api"
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

type TokenManager interface {
	Get(ctx context.Context, logger *slog.Logger, input *api.InputGet) (*keyring.AccessToken, error)
}

// Input contains all the dependencies and configuration needed by the Controller.
// It encapsulates file system access, configuration reading, token generation, and output handling.
// The IsGitCredential flag determines whether to format output for Git's credential helper protocol.
type Input struct {
	ConfigFilePath  string       // Path to the configuration file
	OutputFormat    string       // Output format ("json" or empty for plain text)
	FS              afero.Fs     // File system abstraction for testing
	ConfigReader    ConfigReader // Configuration file reader
	Env             *config.Env  // Environment variable provider
	Stdout          io.Writer    // Output writer
	IsGitCredential bool         // Whether to output in Git credential helper format
	TokenManager    TokenManager // TokenManager for handling token retrieval and creation
}

// NewInput creates a new Input instance with default production values.
// It sets up all necessary dependencies including file system, HTTP client, and keyring access.
func NewInput(configFilePath string) *Input {
	fs := afero.NewOsFs()
	return &Input{
		TokenManager:   api.New(api.NewInput()),
		ConfigFilePath: configFilePath,
		FS:             fs,
		ConfigReader:   config.NewReader(fs),
		Env:            config.NewEnv(os.Getenv, runtime.GOOS),
		Stdout:         os.Stdout,
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
