//nolint:funlen,gocognit,gocritic,nestif
package get

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
)

func TestController_output(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		token           *ghtkn.AccessToken
		outputFormat    string
		isGitCredential bool
		wantOutput      string
		wantErr         bool
	}{
		{
			name: "plain text output",
			token: &ghtkn.AccessToken{
				AccessToken:    "test-token-123",
				ExpirationDate: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
			outputFormat:    "",
			isGitCredential: false,
			wantOutput:      "test-token-123\n",
			wantErr:         false,
		},
		{
			name: "JSON output",
			token: &ghtkn.AccessToken{
				AccessToken:    "test-token-json",
				ExpirationDate: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
			outputFormat:    "json",
			isGitCredential: false,
			wantOutput:      "",
			wantErr:         false,
		},
		{
			name: "Git credential helper output",
			token: &ghtkn.AccessToken{
				AccessToken:    "test-token-git",
				ExpirationDate: time.Time{},
				Login:          "testuser",
			},
			outputFormat:    "",
			isGitCredential: true,
			wantOutput:      "username=testuser\npassword=test-token-git\n\n",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf := &bytes.Buffer{}
			input := &Input{
				OutputFormat:    tt.outputFormat,
				IsGitCredential: tt.isGitCredential,
				Stdout:          buf,
			}
			controller := &Controller{input: input}

			err := controller.output("", tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("output() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				if tt.outputFormat == "json" {
					// Verify it's valid JSON and contains expected fields
					var result map[string]interface{}
					if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
						t.Errorf("output() produced invalid JSON: %v", err)
					}
					if result["access_token"] != tt.token.AccessToken {
						t.Errorf("JSON output missing or incorrect access_token")
					}
					if result["expiration_date"] != tt.token.ExpirationDate.Format(time.RFC3339) {
						t.Errorf("expiration_date: wanted %s, got %s", tt.token.ExpirationDate.Format(time.RFC3339), result["expiration_date"])
					}
				} else {
					if output != tt.wantOutput {
						t.Errorf("output() = %v, want %v", output, tt.wantOutput)
					}
				}
			}
		})
	}
}

func TestController_outputJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    any
		wantErr bool
	}{
		{
			name: "valid data",
			data: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			wantErr: false,
		},
		{
			name: "access token",
			data: &ghtkn.AccessToken{
				AccessToken:    "test-token",
				ExpirationDate: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf := &bytes.Buffer{}
			input := &Input{
				Stdout: buf,
			}
			controller := &Controller{input: input}

			err := controller.outputJSON(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("outputJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				// Verify it's valid JSON
				if !strings.HasPrefix(output, "{") && !strings.HasPrefix(output, "null") {
					t.Errorf("outputJSON() produced invalid JSON output")
				}
				// Verify indentation
				if tt.data != nil && !strings.Contains(output, "\n") {
					t.Error("outputJSON() should produce indented JSON")
				}
			}
		})
	}
}
