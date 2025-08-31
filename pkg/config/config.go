// Package config provides configuration management for ghtkn.
// It handles reading and validating configuration files for GitHub App authentication.
package config

import (
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure for ghtkn.
// It contains settings for persistence and a list of GitHub Apps.
type Config struct {
	Persist bool   `json:"persist,omitempty"`
	Apps    []*App `json:"apps"`
}

// Validate checks if the Config is valid.
// It ensures the config is not nil and contains at least one app.
// It also validates each app in the configuration.
func (cfg *Config) Validate() error {
	if cfg == nil {
		return errors.New("config is required")
	}
	if len(cfg.Apps) == 0 {
		return errors.New("apps is required")
	}
	for _, app := range cfg.Apps {
		if err := app.Validate(); err != nil {
			return fmt.Errorf("app is invalid: %w", slogerr.With(err, "app", app.Name))
		}
	}
	return nil
}

// App represents a GitHub App configuration.
// Each app must have a unique name and a client ID for authentication.
type App struct {
	Name     string `json:"name"`
	ClientID string `json:"client_id" yaml:"client_id"`
	Default  bool   `json:"default,omitempty"`
}

// Validate checks if the App configuration is valid.
// It ensures both Name and ClientID fields are present.
func (app *App) Validate() error {
	if app.Name == "" {
		return errors.New("name is required")
	}
	if app.ClientID == "" {
		return errors.New("client_id is required")
	}
	return nil
}

// Default provides a default configuration template for ghtkn.
// This template can be used to create an initial configuration file.
const Default = `# yaml-language-server: $schema=https://raw.githubusercontent.com/suzuki-shunsuke/ghtkn/refs/heads/main/json-schema/ghtkn.json
# ghtkn - https://github.com/suzuki-shunsuke/ghtkn
persist: true
apps:
  - name: suzuki-shunsuke/write (The name to identify the app)
    client_id: <Your GitHub App Client ID>
    default: true
`

// Reader handles reading configuration files from the filesystem.
type Reader struct {
	fs afero.Fs
}

// NewReader creates a new configuration Reader with the given filesystem.
func NewReader(fs afero.Fs) *Reader {
	return &Reader{fs: fs}
}

// Read reads and parses a configuration file from the given path.
// It decodes the YAML content into the provided Config struct.
// If configFilePath is empty, it returns nil without reading anything.
func (r *Reader) Read(cfg *Config, configFilePath string) error {
	if configFilePath == "" {
		return nil
	}
	f, err := r.fs.Open(configFilePath)
	if err != nil {
		return fmt.Errorf("open a configuration file: %w", err)
	}
	defer f.Close()
	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return fmt.Errorf("decode a configuration file as YAML: %w", err)
	}
	return nil
}
