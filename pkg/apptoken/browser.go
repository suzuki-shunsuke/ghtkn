package apptoken

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/suzuki-shunsuke/go-exec/goexec"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

var errNoCommandFound = errors.New("no command found to open the browser")

func openBrowser(ctx context.Context, url string) error {
	for _, cmd := range cmds() {
		if _, err := exec.LookPath(cmd); err != nil {
			continue
		}
		if err := goexec.Command(ctx, cmd, url).Run(); err != nil {
			return fmt.Errorf("open the browser: %w", slogerr.With(err, "command_to_open_browser", cmd))
		}
		return nil
	}
	return errNoCommandFound
}
