package apptoken

import (
	"context"

	"golang.org/x/sys/windows"
)

// cmds returns the list of commands to try for opening a browser on Windows.
// On Windows, ShellExecute is used directly, so no commands are needed.
func cmds() []string {
	return nil
}

// openB opens a browser on Windows using the ShellExecute API.
// It directly calls the Windows API to open the default browser.
func openB(_ context.Context, url string) error {
	return windows.ShellExecute(0, nil, windows.StringToUTF16Ptr(url), nil, nil, windows.SW_SHOWNORMAL)
}
