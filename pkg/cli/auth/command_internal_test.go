package auth

import (
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
)

func TestNewInputAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          *Args
		clipboardSet  bool
		wantAppName   string
		wantConfig    string
		wantClipboard *bool
	}{
		{
			// An empty app name means "not given", leaving the SDK to fall back to
			// GHTKN_APP or the default app. --clipboard defaults to false, so without it
			// being set the override must stay nil rather than carry that default, and
			// the SDK resolves GHTKN_CLIPBOARD and the config itself.
			name: "nothing given",
			args: &Args{GlobalFlags: &flag.GlobalFlags{}},
		},
		{
			name:        "app name and config path are passed through",
			args:        &Args{GlobalFlags: &flag.GlobalFlags{Config: "/path/to/ghtkn.yaml"}, AppName: "my-app"},
			wantAppName: "my-app",
			wantConfig:  "/path/to/ghtkn.yaml",
		},
		{
			name:          "--clipboard sets the override",
			args:          &Args{GlobalFlags: &flag.GlobalFlags{}, Clipboard: true},
			clipboardSet:  true,
			wantClipboard: new(true),
		},
		{
			// --clipboard=false must reach the SDK rather than being dropped as a zero
			// value, so that it overrides GHTKN_CLIPBOARD and the config.
			name:          "--clipboard=false sets the override too",
			args:          &Args{GlobalFlags: &flag.GlobalFlags{}},
			clipboardSet:  true,
			wantClipboard: new(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := newInputAuth(tt.args, tt.clipboardSet)
			if input.AppName != tt.wantAppName {
				t.Errorf("AppName = %q, want %q", input.AppName, tt.wantAppName)
			}
			if input.ConfigFilePath != tt.wantConfig {
				t.Errorf("ConfigFilePath = %q, want %q", input.ConfigFilePath, tt.wantConfig)
			}
			switch {
			case tt.wantClipboard == nil:
				if input.Clipboard != nil {
					t.Errorf("Clipboard = %v, want nil", *input.Clipboard)
				}
			case input.Clipboard == nil:
				t.Errorf("Clipboard = nil, want %v", *tt.wantClipboard)
			case *input.Clipboard != *tt.wantClipboard:
				t.Errorf("Clipboard = %v, want %v", *input.Clipboard, *tt.wantClipboard)
			}
		})
	}
}
