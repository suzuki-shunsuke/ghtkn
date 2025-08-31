package apptoken

import "context"

func cmds() []string {
	return []string{"xdg-open", "x-www-browser", "www-browser"}
}

func openB(ctx context.Context, url string) error {
	return runCmd(ctx, url)
}
