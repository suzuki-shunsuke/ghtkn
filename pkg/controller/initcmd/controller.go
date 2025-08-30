package initcmd

import (
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
)

type Controller struct {
	fs  afero.Fs
	env *config.Env
}

func New(fs afero.Fs, env *config.Env) *Controller {
	return &Controller{
		fs:  fs,
		env: env,
	}
}
