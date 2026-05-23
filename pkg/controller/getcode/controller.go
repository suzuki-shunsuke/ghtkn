package getcode

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
)

type Controller struct {
	input *Input
}

func New(input *Input) *Controller {
	return &Controller{
		input: input,
	}
}

type Client interface {
	Get(ctx context.Context, logger *slog.Logger, input *ghtkn.InputGet) (*ghtkn.AccessToken, *ghtkn.AppConfig, error)
}

type Input struct {
	OutputFormat    string    // Output format ("json" or empty for plain text)
	Stdout          io.Writer // Output writer
	IsGitCredential bool      // Whether to output in Git credential helper format
	Client          Client
}

func NewInput() *Input {
	return &Input{
		Stdout: os.Stdout,
		Client: ghtkn.New(),
	}
}
