// Package clipboard copies the device flow one-time code to the system clipboard.
// It implements github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow.CopyTextToClipboard
// so the ghtkn CLI can inject it into the SDK via Client.SetCopyOnetimeCodeToClipboard.
// Keeping golang.design/x/clipboard here, rather than in the SDK, avoids forcing its
// large transitive dependency tree onto every SDK consumer.
package clipboard

import (
	"context"
	"fmt"
	"sync"

	"golang.design/x/clipboard"
)

// New returns a function that copies text to the system clipboard. It satisfies the
// SDK's deviceflow.CopyTextToClipboard signature. clipboard.Init touches platform
// clipboard APIs (Obj-C/purego on macOS, X11 on Linux/BSD, syscalls on Windows), so
// it runs at most once and its error is reused on later calls. The context is accepted
// to match the signature; the underlying write is synchronous and not cancellable.
func New() func(ctx context.Context, text string) error {
	var once sync.Once
	var initErr error
	return func(_ context.Context, text string) error {
		once.Do(func() {
			initErr = clipboard.Init()
		})
		if initErr != nil {
			return fmt.Errorf("initialize the clipboard: %w", initErr)
		}
		clipboard.Write(clipboard.FmtText, []byte(text))
		return nil
	}
}
