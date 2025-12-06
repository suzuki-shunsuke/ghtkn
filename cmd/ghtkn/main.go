package main

import (
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
)

var version = ""

func main() {
	urfave.Main("ghtkn", version, cli.Run)
}
