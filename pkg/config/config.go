package config

import (
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Persist bool   `json:"persist,omitempty"`
	Apps    []*App `json:"apps"`
}

func (cfg *Config) Validate() error {
	if cfg == nil {
		return errors.New("config is required")
	}
	if len(cfg.Apps) == 0 {
		return errors.New("apps is required")
	}
	for _, app := range cfg.Apps {
		if err := app.Validate(); err != nil {
			return fmt.Errorf("app is invalid: %w", slogerr.With(err, "app", app.ID))
		}
	}
	return nil
}

type App struct {
	ID       string `json:"id"`
	ClientID string `json:"client_id" yaml:"client_id"`
	Default  bool   `json:"default,omitempty"`
}

func (app *App) Validate() error {
	if app.ID == "" {
		return errors.New("id is required")
	}
	if app.ClientID == "" {
		return errors.New("client_id is required")
	}
	return nil
}

const Default = `# yaml-language-server: $schema=https://raw.githubusercontent.com/suzuki-shunsuke/ghtkn/refs/heads/main/json-schema/ghtkn.json
# ghtkn - https://github.com/suzuki-shunsuke/ghtkn
persist: true
apps:
  - id: suzuki-shunsuke/write (The name to identify the app)
    client_id: <Your GitHub App Client ID>
    default: true
`

type Reader struct {
	fs afero.Fs
}

func NewReader(fs afero.Fs) *Reader {
	return &Reader{fs: fs}
}

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
