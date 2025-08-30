package get

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/apptoken"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

type Controller struct {
	input *Input
}

func New(input *Input) *Controller {
	return &Controller{
		input: input,
	}
}

type Input struct {
	ConfigFilePath string
	OutputFormat   string
	MinExpiration  time.Duration
	FS             afero.Fs
	ConfigReader   ConfigReader
	Env            *config.Env
	AppTokenClient AppTokenClient
	Stdout         io.Writer
	Keyring        Keyring
	Now            func() time.Time
}

func NewInput(configFilePath string) *Input {
	fs := afero.NewOsFs()
	return &Input{
		ConfigFilePath: configFilePath,
		FS:             fs,
		ConfigReader:   config.NewReader(fs),
		Env:            config.NewEnv(os.Getenv),
		AppTokenClient: apptoken.NewClient(http.DefaultClient),
		Stdout:         os.Stdout,
		Keyring:        keyring.New("github.com/suzuki-shunsuke/ghtkn"),
		Now:            time.Now,
	}
}

func (i *Input) IsJSON() bool {
	return i.OutputFormat == "json"
}

func (i *Input) Validate() error {
	if i.OutputFormat != "" && i.OutputFormat != "json" {
		return errors.New("output format must be empty or 'json'")
	}
	return nil
}

type ConfigReader interface {
	Read(cfg *config.Config, configFilePath string) error
}

type AppTokenClient interface {
	Create(ctx context.Context, logger *slog.Logger, clientID string) (*apptoken.AccessToken, error)
}

type Keyring interface {
	Get(key string) (*keyring.AccessToken, error)
	Set(key string, token *keyring.AccessToken) error
}
