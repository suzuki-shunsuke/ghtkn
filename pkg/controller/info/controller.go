package info

import "io"

type Controller struct {
	getEnv func(string) string
	stdout io.Writer
}

func New(stdout io.Writer, getEnv func(string) string) *Controller {
	return &Controller{
		stdout: stdout,
		getEnv: getEnv,
	}
}
