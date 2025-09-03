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
	"github.com/suzuki-shunsuke/ghtkn/pkg/github"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

func TestController_Run(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
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
					MinExpiration:  time.Hour,
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
					Stdout:       &bytes.Buffer{},
					Keyring:      &mockKeyring{},
					Now:          func() time.Time { return fixedTime },
					NewGitHub: func(ctx context.Context, token string) api.GitHub {
						return api.NewMockGitHub(&github.User{
							Login: "test-user",
						}, nil)(ctx, token)
					},
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
					MinExpiration:  time.Hour,
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
					TokenManager: api.New(api.NewInput()),
					// AppTokenClient: &mockAppTokenClient{
					// 	token: &apptoken.AccessToken{
					// 		AccessToken:    "new-token",
					// 		ExpirationDate: keyring.FormatDate(futureTime),
					// 	},
					// },
					Stdout: &bytes.Buffer{},
					Keyring: &mockKeyring{
						tokens: map[string]*keyring.AccessToken{
							"test-client-id": {
								App:            "test-app",
								AccessToken:    "cached-token",
								ExpirationDate: keyring.FormatDate(futureTime),
								Login:          "cached-user",
							},
						},
					},
					Now: func() time.Time { return fixedTime },
					NewGitHub: api.NewMockGitHub(&github.User{
						Login: "cached-user",
					}, nil),
				}
			},
			wantErr:    false,
			wantOutput: "cached-token\n",
		},
		{
			name: "expired token in keyring triggers new token creation",
			setupInput: func() *get.Input {
				expiredTime := fixedTime.Add(30 * time.Minute)
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					MinExpiration:  time.Hour,
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
					TokenManager: api.New(api.NewInput()),
					// AppTokenClient: &mockAppTokenClient{
					// 	token: &apptoken.AccessToken{
					// 		AccessToken:    "new-token",
					// 		ExpirationDate: keyring.FormatDate(futureTime),
					// 	},
					// },
					Stdout: &bytes.Buffer{},
					Keyring: &mockKeyring{
						tokens: map[string]*keyring.AccessToken{
							"test-client-id": {
								App:            "test-app",
								AccessToken:    "expired-token",
								ExpirationDate: keyring.FormatDate(expiredTime),
							},
						},
					},
					Now: func() time.Time { return fixedTime },
					NewGitHub: api.NewMockGitHub(&github.User{
						Login: "cached-user",
					}, nil),
				}
			},
			wantErr:      false,
			wantOutput:   "new-token\n",
			checkKeyring: true,
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
					MinExpiration:  time.Hour,
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
					Stdout:  &bytes.Buffer{},
					Keyring: &mockKeyring{},
					Now:     func() time.Time { return fixedTime },
					NewGitHub: api.NewMockGitHub(&github.User{
						Login: "cached-user",
					}, nil),
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
					MinExpiration:  time.Hour,
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
					Stdout:    &bytes.Buffer{},
					Keyring:   &mockKeyring{},
					Now:       func() time.Time { return fixedTime },
					NewGitHub: api.NewMockGitHub(nil, errors.New("GitHub API error")),
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
					MinExpiration:  time.Hour,
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
					TokenManager: api.New(api.NewInput()),
					// AppTokenClient: &mockAppTokenClient{
					// 	token: &apptoken.AccessToken{
					// 		AccessToken:    "new-token",
					// 		ExpirationDate: keyring.FormatDate(futureTime),
					// 	},
					// },
					Stdout: &bytes.Buffer{},
					Keyring: &mockKeyring{
						tokens: map[string]*keyring.AccessToken{
							"test-client-id": {
								App:            "test-app",
								AccessToken:    "cached-token",
								ExpirationDate: keyring.FormatDate(futureTime),
								// Login is empty, will trigger GetUser call
							},
						},
					},
					Now:       func() time.Time { return fixedTime },
					NewGitHub: api.NewMockGitHub(nil, errors.New("GitHub API rate limit error")),
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
					MinExpiration:  time.Hour,
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
					// 		AccessToken:    "test-token-json",
					// 		ExpirationDate: keyring.FormatDate(futureTime),
					// 	},
					// },
					Stdout:  &bytes.Buffer{},
					Keyring: &mockKeyring{},
					Now:     func() time.Time { return fixedTime },
					NewGitHub: api.NewMockGitHub(&github.User{
						Login: "cached-user",
					}, nil),
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

			if tt.checkKeyring {
				kr := input.Keyring.(*mockKeyring)
				if kr.tokens["test-client-id"] == nil {
					t.Error("Token was not stored in keyring")
				} else if kr.tokens["test-client-id"].AccessToken != "new-token" {
					t.Errorf("Stored token = %v, want new-token", kr.tokens["test-client-id"].AccessToken)
				}
			}
		})
	}
}
