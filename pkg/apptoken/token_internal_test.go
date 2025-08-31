package apptoken

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestClient_getDeviceCode(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()
	tests := []struct {
		name        string
		clientID    string
		handler     http.HandlerFunc
		want        *DeviceCodeResponse
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful device code request",
			clientID: "test-client-id",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/login/device/code" {
					t.Errorf("expected /login/device/code, got %s", r.URL.Path)
				}

				body, _ := io.ReadAll(r.Body)
				var req map[string]string
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("failed to unmarshal request body: %v", err)
				}
				if req["client_id"] != "test-client-id" {
					t.Errorf("expected client_id test-client-id, got %s", req["client_id"])
				}

				resp := DeviceCodeResponse{
					DeviceCode:      "device123",
					UserCode:        "USER-CODE",
					VerificationURI: "https://github.com/login/device",
					ExpiresIn:       900,
					Interval:        5,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp) //nolint:errchkjson,errcheck
			},
			want: &DeviceCodeResponse{
				DeviceCode:      "device123",
				UserCode:        "USER-CODE",
				VerificationURI: "https://github.com/login/device",
				ExpiresIn:       900,
				Interval:        5,
			},
			wantErr: false,
		},
		{
			name:     "error response from GitHub",
			clientID: "test-client-id",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"invalid_client","error_description":"The client_id is not valid"}`)) //nolint:errcheck
			},
			want:        nil,
			wantErr:     true,
			errContains: "error from GitHub",
		},
		{
			name:     "invalid JSON response",
			clientID: "test-client-id",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`invalid json`)) //nolint:errcheck
			},
			want:        nil,
			wantErr:     true,
			errContains: "unmarshal response body as JSON",
		},
		{
			name:     "empty client ID",
			clientID: "",
			handler: func(_ http.ResponseWriter, _ *http.Request) {
				// Should not be called
				t.Error("handler should not be called with empty client ID")
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			// Override the GitHub API URL in the test
			originalURL := "https://github.com/login/device/code"
			_ = originalURL // We'll need to modify the actual implementation to make URL configurable

			input := NewMockInput()
			input.HTTPClient = server.Client()
			client := &Client{
				input: input,
			}

			// Since we can't easily override the URL, we'll use a different approach
			// We need to modify the actual request URL
			ctx := t.Context()

			// Create a custom transport that redirects requests
			transport := &testTransport{
				server: server,
				base:   http.DefaultTransport,
			}
			client.input.HTTPClient = &http.Client{Transport: transport}

			got, err := client.getDeviceCode(ctx, tt.clientID)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			if tt.wantErr {
				t.Fatalf("expected error but got nil")
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("DeviceCodeResponse mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClient_checkAccessToken(t *testing.T) { //nolint:gocognit,cyclop,funlen
	t.Parallel()
	tests := []struct {
		name       string
		clientID   string
		deviceCode string
		handler    http.HandlerFunc
		want       *AccessTokenResponse
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "successful token response",
			clientID:   "test-client-id",
			deviceCode: "device123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/login/oauth/access_token" {
					t.Errorf("expected /login/oauth/access_token, got %s", r.URL.Path)
				}

				body, _ := io.ReadAll(r.Body)
				var req map[string]string
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("failed to unmarshal request body: %v", err)
				}

				if req["client_id"] != "test-client-id" {
					t.Errorf("expected client_id test-client-id, got %s", req["client_id"])
				}
				if req["device_code"] != "device123" {
					t.Errorf("expected device_code device123, got %s", req["device_code"])
				}
				if req["grant_type"] != "urn:ietf:params:oauth:grant-type:device_code" {
					t.Errorf("unexpected grant_type: %s", req["grant_type"])
				}

				resp := AccessTokenResponse{
					AccessToken: "gho_testtoken123",
					ExpiresIn:   28800,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp) //nolint:errchkjson,errcheck
			},
			want: &AccessTokenResponse{
				AccessToken: "gho_testtoken123",
				ExpiresIn:   28800,
			},
			wantErr: false,
		},
		{
			name:       "authorization pending",
			clientID:   "test-client-id",
			deviceCode: "device123",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				resp := AccessTokenResponse{
					Error: "authorization_pending",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp) //nolint:errchkjson,errcheck
			},
			want:    nil,
			wantErr: true,
			errMsg:  "authorization_pending",
		},
		{
			name:       "slow down response",
			clientID:   "test-client-id",
			deviceCode: "device123",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				resp := AccessTokenResponse{
					Error: "slow_down",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp) //nolint:errcheck,errchkjson
			},
			want:    nil,
			wantErr: true,
			errMsg:  "slow_down",
		},
		{
			name:       "access denied",
			clientID:   "test-client-id",
			deviceCode: "device123",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				resp := AccessTokenResponse{
					Error: "access_denied",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp) //nolint:errcheck,errchkjson
			},
			want:    nil,
			wantErr: true,
			errMsg:  "access_denied",
		},
		{
			name:       "empty response",
			clientID:   "test-client-id",
			deviceCode: "device123",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{}`)) //nolint:errcheck
			},
			want:    nil,
			wantErr: true,
			errMsg:  "unexpected response: {}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			transport := &testTransport{
				server: server,
				base:   http.DefaultTransport,
			}

			input := NewMockInput()
			input.HTTPClient = &http.Client{Transport: transport}
			client := NewClient(input)

			got, err := client.checkAccessToken(t.Context(), tt.clientID, tt.deviceCode)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if tt.wantErr {
				t.Fatalf("expected error but got nil")
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("AccessTokenResponse mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClient_pollForAccessToken(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name        string
		clientID    string
		deviceCode  *DeviceCodeResponse
		handler     http.HandlerFunc
		want        *AccessTokenResponse
		wantErr     bool
		errContains string
		timeout     time.Duration
	}{
		{
			name:     "successful after one poll",
			clientID: "test-client-id",
			deviceCode: &DeviceCodeResponse{
				DeviceCode:      "device123",
				UserCode:        "USER-CODE",
				VerificationURI: "https://github.com/login/device",
				ExpiresIn:       10,
				Interval:        1,
			},
			handler: func() http.HandlerFunc {
				callCount := 0
				return func(w http.ResponseWriter, _ *http.Request) {
					callCount++
					if callCount == 1 {
						// First call returns pending
						resp := AccessTokenResponse{
							Error: "authorization_pending",
						}
						json.NewEncoder(w).Encode(resp) //nolint:errcheck
					} else {
						// Second call returns success
						resp := AccessTokenResponse{
							AccessToken: "gho_testtoken123",
							ExpiresIn:   28800,
						}
						json.NewEncoder(w).Encode(resp) //nolint:errcheck
					}
				}
			}(),
			want: &AccessTokenResponse{
				AccessToken: "gho_testtoken123",
				ExpiresIn:   28800,
			},
			wantErr: false,
			timeout: 5 * time.Second,
		},
		{
			name:     "context cancelled",
			clientID: "test-client-id",
			deviceCode: &DeviceCodeResponse{
				DeviceCode:      "device123",
				UserCode:        "USER-CODE",
				VerificationURI: "https://github.com/login/device",
				ExpiresIn:       10,
				Interval:        1,
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				resp := AccessTokenResponse{
					Error: "authorization_pending",
				}
				json.NewEncoder(w).Encode(resp) //nolint:errcheck,errchkjson
			},
			want:        nil,
			wantErr:     true,
			errContains: "context was cancelled",
			timeout:     5 * time.Millisecond,
		},
		// TODO
		// {
		// 	name:     "slow down handling",
		// 	clientID: "test-client-id",
		// 	deviceCode: &DeviceCodeResponse{
		// 		DeviceCode:      "device123",
		// 		UserCode:        "USER-CODE",
		// 		VerificationURI: "https://github.com/login/device",
		// 		ExpiresIn:       10,
		// 		Interval:        1,
		// 	},
		// 	handler: func() http.HandlerFunc {
		// 		callCount := 0
		// 		return func(w http.ResponseWriter, r *http.Request) {
		// 			callCount++
		// 			if callCount == 1 {
		// 				// First call returns slow_down
		// 				resp := AccessTokenResponse{
		// 					Error: "slow_down",
		// 				}
		// 				json.NewEncoder(w).Encode(resp)
		// 			} else {
		// 				// Subsequent calls return success
		// 				resp := AccessTokenResponse{
		// 					AccessToken: "gho_testtoken123",
		// 					ExpiresIn:   28800,
		// 				}
		// 				json.NewEncoder(w).Encode(resp)
		// 			}
		// 		}
		// 	}(),
		// 	want: &AccessTokenResponse{
		// 		AccessToken: "gho_testtoken123",
		// 		ExpiresIn:   28800,
		// 	},
		// 	wantErr: false,
		// 	timeout: 10 * time.Second,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			transport := &testTransport{
				server: server,
				base:   http.DefaultTransport,
			}

			input := NewMockInput()
			input.HTTPClient = &http.Client{Transport: transport}
			client := NewClient(input)

			ctx := t.Context()
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			}

			got, err := client.pollForAccessToken(ctx, tt.clientID, tt.deviceCode)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			if tt.wantErr {
				t.Fatalf("expected error but got nil")
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("AccessTokenResponse mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClient_Create(t *testing.T) { //nolint:gocognit,cyclop,funlen
	t.Parallel()
	tests := []struct {
		name        string
		clientID    string
		handler     http.HandlerFunc
		want        *AccessToken
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty client ID",
			clientID:    "",
			handler:     nil,
			want:        nil,
			wantErr:     true,
			errContains: "client id is required",
		},
		{
			name:     "successful flow",
			clientID: "test-client-id",
			handler: func() http.HandlerFunc {
				callCount := 0
				return func(w http.ResponseWriter, r *http.Request) {
					callCount++
					switch r.URL.Path {
					case "/login/device/code":
						// Device code request
						resp := DeviceCodeResponse{
							DeviceCode:      "device123",
							UserCode:        "USER-CODE",
							VerificationURI: "https://github.com/login/device",
							ExpiresIn:       10,
							Interval:        1,
						}
						json.NewEncoder(w).Encode(resp) //nolint:errcheck
					case "/login/oauth/access_token":
						// Access token request
						if callCount <= 2 {
							// First call returns pending
							resp := AccessTokenResponse{
								Error: "authorization_pending",
							}
							json.NewEncoder(w).Encode(resp) //nolint:errcheck
							return
						}
						// Second call returns success
						resp := AccessTokenResponse{
							AccessToken: "gho_testtoken123",
							ExpiresIn:   28800,
						}
						json.NewEncoder(w).Encode(resp) //nolint:errcheck
					}
				}
			}(),
			want: &AccessToken{
				AccessToken: "gho_testtoken123",
				// ExpirationDate will be set dynamically
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var server *httptest.Server
			var transport *testTransport

			if tt.handler != nil {
				server = httptest.NewServer(tt.handler)
				defer server.Close()

				transport = &testTransport{
					server: server,
					base:   http.DefaultTransport,
				}
			}

			fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			input := NewMockInput()
			input.HTTPClient = &http.Client{Transport: transport}
			input.Now = func() time.Time { return fixedTime }
			input.Stderr = &bytes.Buffer{}
			client := NewClient(input)

			logger := slog.New(slog.DiscardHandler)

			got, err := client.Create(t.Context(), logger, tt.clientID)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if tt.wantErr {
				t.Fatalf("expected error but got nil")
				return
			}

			// Compare without ExpirationDate first
			if got.AccessToken != tt.want.AccessToken {
				t.Errorf("AccessToken = %v, want %v", got.AccessToken, tt.want.AccessToken)
			}

			// Check that ExpirationDate is set
			if got.ExpirationDate == "" {
				t.Error("ExpirationDate should be set")
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	t.Parallel()
	httpClient := &http.Client{}
	input := NewMockInput()
	input.HTTPClient = httpClient
	client := NewClient(input)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.input.HTTPClient != httpClient {
		t.Error("httpClient not set correctly")
	}

	if client.input.Now == nil {
		t.Error("now function not set")
	}

	if client.input.Stderr == nil {
		t.Error("stderr not set")
	}
}

// testTransport is a custom transport that redirects GitHub API requests to our test server
type testTransport struct {
	server *httptest.Server
	base   http.RoundTripper
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect GitHub API requests to our test server
	if strings.Contains(req.URL.Host, "github.com") {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(t.server.URL, "http://")
	}
	return t.base.RoundTrip(req) //nolint:wrapcheck
}

func TestErrNoCommandFound(t *testing.T) {
	t.Parallel()
	if errNoCommandFound == nil {
		t.Fatal("errNoCommandFound should not be nil")
	}

	if !errors.Is(errNoCommandFound, errNoCommandFound) {
		t.Error("errNoCommandFound should be comparable with errors.Is")
	}
}
