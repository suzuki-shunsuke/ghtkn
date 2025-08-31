package apptoken

import "context"

func cmds() []string {
	return []string{"open"}
}

func openB(ctx context.Context, url string) error {
	return runCmd(ctx, url)
}
