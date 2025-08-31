package apptoken

import "context"

// cmds returns the list of commands to try for opening a browser on Linux.
// It tries xdg-open first, then falls back to x-www-browser and www-browser.
func cmds() []string {
	return []string{"xdg-open", "x-www-browser", "www-browser"}
}

// openB opens a browser on Linux using the appropriate system command.
// It delegates to runCmd with the platform-specific commands.
func openB(ctx context.Context, url string) error {
	return runCmd(ctx, url)
}
