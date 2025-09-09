//nolint:forcetypeassert,funlen
package get_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
)

func TestController_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupInput   func() *get.Input
		wantErr      bool
		wantOutput   string
		checkKeyring bool
	}{
		{
			name: "successful token creation without persistence",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					Env:            &config.Env{App: "test-app"},
					Stdout:         &bytes.Buffer{},
				}
			},
			wantErr:    false,
			wantOutput: "test-token-123\n",
		},
		{
			name: "token creation error",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					Env:            &config.Env{App: "test-app"},
					Stdout:         &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "JSON output format",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "json",
					Env:            &config.Env{App: "test-app"},
					Stdout:         &bytes.Buffer{},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := tt.setupInput()
			controller := get.New(input)
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			err := controller.Run(ctx, logger, &ghtkn.InputGet{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && input.OutputFormat != "json" {
				output := input.Stdout.(*bytes.Buffer).String()
				if output != tt.wantOutput {
					t.Errorf("Run() output = %v, want %v", output, tt.wantOutput)
				}
			}
		})
	}
}
