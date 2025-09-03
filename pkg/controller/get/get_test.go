//nolint:forcetypeassert,funlen,maintidx
package get_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/api"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

func TestController_Run(t *testing.T) {
	t.Parallel()

	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

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
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: false,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env: &config.Env{App: "test-app"},
					TokenManager: api.NewMockTokenManager(&keyring.AccessToken{
						AccessToken:    "test-token-123",
						ExpirationDate: keyring.FormatDate(futureTime),
						Login:          "test-user",
					}, nil),
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr:    false,
			wantOutput: "test-token-123\n",
		},
		{
			name: "successful token retrieval from keyring",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: true,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env: &config.Env{App: "test-app"},
					TokenManager: api.NewMockTokenManager(&keyring.AccessToken{
						AccessToken:    "cached-token",
						ExpirationDate: keyring.FormatDate(futureTime),
						Login:          "cached-user",
					}, nil),
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr:    false,
			wantOutput: "cached-token\n",
		},
		{
			name: "expired token in keyring triggers new token creation",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: true,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env: &config.Env{App: "test-app"},
					TokenManager: api.NewMockTokenManager(&keyring.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: keyring.FormatDate(futureTime),
						Login:          "test-user",
					}, nil),
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr:    false,
			wantOutput: "new-token\n",
		},
		{
			name: "config read error",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						err: errors.New("config read error"),
					},
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "invalid config",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Apps: []*config.App{}, // No apps configured
						},
					},
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "token creation error",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: false,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env:          &config.Env{App: "test-app"},
					TokenManager: api.New(api.NewInput()),
					// AppTokenClient: &mockAppTokenClient{
					// 	err: errors.New("token creation failed"),
					// },
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "GitHub API GetUser error",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: false,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env:          &config.Env{App: "test-app"},
					TokenManager: api.New(api.NewInput()),
					// AppTokenClient: &mockAppTokenClient{
					// 	token: &apptoken.AccessToken{
					// 		AccessToken:    "test-token-123",
					// 		ExpirationDate: keyring.FormatDate(futureTime),
					// 	},
					// },
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "cached token without login and GitHub API error",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: true,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env:          &config.Env{App: "test-app"},
					TokenManager: api.NewMockTokenManager(nil, errors.New("GitHub API rate limit error")),
					Stdout:       &bytes.Buffer{},
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
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: false,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env: &config.Env{App: "test-app"},
					TokenManager: api.NewMockTokenManager(&keyring.AccessToken{
						AccessToken:    "test-token-json",
						ExpirationDate: keyring.FormatDate(futureTime),
						Login:          "test-user",
					}, nil),
					Stdout: &bytes.Buffer{},
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

			err := controller.Run(ctx, logger)
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
