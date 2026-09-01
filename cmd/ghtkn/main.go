package main

import (
	"github.com/suzuki-shunsuke/cobra-util/cobrautil"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli"
)

var version = ""

func main() {
	cobrautil.Main("ghtkn", version, cli.Run)
}
