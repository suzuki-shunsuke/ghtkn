package apptoken

import "context"

// cmds returns the list of commands to try for opening a browser on macOS.
// On macOS, the standard command is "open".
func cmds() []string {
	return []string{"open"}
}

// openB opens a browser on macOS using the appropriate system command.
// It delegates to runCmd with the platform-specific commands.
func openB(ctx context.Context, url string) error {
	return runCmd(ctx, url)
}
