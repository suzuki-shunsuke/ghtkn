package apptoken

import (
	"context"

	"golang.org/x/sys/windows"
)

func cmds() []string {
	return nil
}

func openB(_ context.Context, url string) error {
	return windows.ShellExecute(0, nil, windows.StringToUTF16Ptr(url), nil, nil, windows.SW_SHOWNORMAL)
}
