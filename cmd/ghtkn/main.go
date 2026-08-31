package main

import (
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cobrautil"
)

var version = ""

func main() {
	cobrautil.Main("ghtkn", version, cli.Run)
}
