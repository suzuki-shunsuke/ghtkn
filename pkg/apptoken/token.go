// Package apptoken handles GitHub App access token generation using OAuth device flow.
// It provides functionality to authenticate GitHub Apps and obtain access tokens.
package apptoken

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// Client handles GitHub App authentication and access token generation.
type Client struct {
	input *Input
}

type Browser interface {
	Open(ctx context.Context, url string) error
}

type Input struct {
	HTTPClient   *http.Client
	Now          func() time.Time
	Stderr       io.Writer
	Browser      Browser
	NewTicker    func(d time.Duration) *time.Ticker
	Logger       *Logger
	DeviceCodeUI DeviceCodeUI
}

type Logger struct {
	FailedToOpenBrowser func(logger *slog.Logger, err error)
}

func NewLogger() *Logger {
	return &Logger{
		FailedToOpenBrowser: func(logger *slog.Logger, err error) {
			slogerr.WithError(logger, err).Warn("failed to open the browser")
		},
	}
}

func NewInput() *Input {
	return &Input{
		HTTPClient:   http.DefaultClient,
		Now:          time.Now,
		Stderr:       os.Stderr,
		Browser:      NewBrowser(),
		NewTicker:    time.NewTicker,
		Logger:       NewLogger(),
		DeviceCodeUI: NewDeviceCodeUI(os.Stderr),
	}
}

func NewMockInput() *Input {
	return &Input{
		HTTPClient: http.DefaultClient,
		Now:        time.Now,
		Stderr:     io.Discard,
		Browser:    NewMockBrowser(nil),
		NewTicker: func(_ time.Duration) *time.Ticker {
			return time.NewTicker(10 * time.Millisecond) //nolint:mnd
		},
		Logger:       NewLogger(),
		DeviceCodeUI: &MockDeviceCodeUI{},
	}
}

// NewClient creates a new Client with the provided HTTP client.
// The client uses the provided HTTP client for all API requests.
func NewClient(input *Input) *Client {
	return &Client{
		input: input,
	}
}

// DeviceCodeResponse represents the response from GitHub's device code endpoint.
// It contains the device code and user code needed for authentication.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// AccessTokenResponse represents the response from GitHub's access token endpoint.
// It contains either an access token or an error message.
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`

	Error string `json:"error"`
}

// AccessToken represents a GitHub App access token with its metadata.
// It includes the token value, associated app, and expiration date.
type AccessToken struct {
	App            string `json:"app"`
	AccessToken    string `json:"access_token"`
	ExpirationDate string `json:"expiration_date"`
}

// Create initiates the OAuth device flow and returns an access token.
// It displays the verification URL and user code, optionally opens a browser,
// and polls for the access token until the user completes authentication.
func (c *Client) Create(ctx context.Context, logger *slog.Logger, clientID string) (*AccessToken, error) {
	if clientID == "" {
		return nil, errors.New("client id is required")
	}
	deviceCode, err := c.getDeviceCode(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("get device code: %w", err)
	}

	deviceCodeExpirationDate := c.input.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)
	c.input.DeviceCodeUI.Show(deviceCode, deviceCodeExpirationDate)
	if err := c.input.Browser.Open(ctx, deviceCode.VerificationURI); err != nil {
		if !errors.Is(err, errNoCommandFound) {
			c.input.Logger.FailedToOpenBrowser(logger, err)
		}
	}

	token, err := c.pollForAccessToken(ctx, clientID, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	now := c.input.Now()

	return &AccessToken{
		AccessToken:    token.AccessToken,
		ExpirationDate: keyring.FormatDate(now.Add(time.Duration(token.ExpiresIn) * time.Second)),
	}, nil
}

type DeviceCodeUI interface {
	Show(deviceCode *DeviceCodeResponse, expirationDate time.Time)
}

type MockDeviceCodeUI struct{}

func (m *MockDeviceCodeUI) Show(_ *DeviceCodeResponse, _ time.Time) {}

type SimpleDeviceCodeUI struct {
	stderr io.Writer
}

func NewDeviceCodeUI(stderr io.Writer) *SimpleDeviceCodeUI {
	return &SimpleDeviceCodeUI{
		stderr: stderr,
	}
}

func (d *SimpleDeviceCodeUI) Show(deviceCode *DeviceCodeResponse, expirationDate time.Time) {
	fmt.Fprintf(d.stderr, "Please visit: %s\n", deviceCode.VerificationURI)
	fmt.Fprintf(d.stderr, "And enter code: %s\n", deviceCode.UserCode)
	fmt.Fprintf(d.stderr, "Expiration date: %s\n", expirationDate.Format(time.RFC3339))
}

// getDeviceCode requests a device code from GitHub's OAuth device endpoint.
// It returns the device code response containing the user code and verification URL.
func (c *Client) getDeviceCode(ctx context.Context, clientID string) (*DeviceCodeResponse, error) {
	if clientID == "" {
		return nil, errors.New("client id is required")
	}
	jsonData, err := json.Marshal(map[string]string{
		"client_id": clientID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal a request body as JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/device/code", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create a request for device code: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.input.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send a request for device code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, slogerr.With( //nolint:wrapcheck
			errors.New("error from GitHub"),
			"status_code", resp.StatusCode,
			"body", string(body))
	}

	deviceCode := &DeviceCodeResponse{}
	if err := json.Unmarshal(body, deviceCode); err != nil {
		return nil, fmt.Errorf("unmarshal response body as JSON: %w", err)
	}

	return deviceCode, nil
}

// additionalInterval is the minimum polling interval to avoid rate limiting.
const additionalInterval = 5 * time.Second

// pollForAccessToken continuously polls GitHub for an access token.
// It respects the polling interval and handles authorization pending and slow down responses.
// The polling continues until the device code expires or the user completes authentication.
func (c *Client) pollForAccessToken(ctx context.Context, clientID string, deviceCode *DeviceCodeResponse) (*AccessTokenResponse, error) {
	interval := time.Duration(deviceCode.Interval) * time.Second
	if interval < additionalInterval {
		interval = additionalInterval
	}

	ticker := c.input.NewTicker(interval)
	defer ticker.Stop()

	deadline := c.input.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context was cancelled: %w", ctx.Err())
		case <-ticker.C:
			if c.input.Now().After(deadline) {
				return nil, errors.New("device code expired")
			}

			token, err := c.checkAccessToken(ctx, clientID, deviceCode.DeviceCode)
			if err != nil {
				if err.Error() == "authorization_pending" {
					continue
				}
				if err.Error() == "slow_down" {
					ticker.Reset(interval + 5*time.Second)
					continue
				}
				return nil, err
			}

			if token != nil {
				return token, nil
			}
		}
	}
}

// checkAccessToken checks if an access token is available for the given device code.
// It returns the access token if available, or an error indicating the current status.
func (c *Client) checkAccessToken(ctx context.Context, clientID, deviceCode string) (*AccessTokenResponse, error) {
	reqBody := map[string]string{
		"client_id":   clientID,
		"device_code": deviceCode,
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body as JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create a request for access token: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.input.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send a request for access token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	token := &AccessTokenResponse{}
	if err := json.Unmarshal(body, token); err != nil {
		return nil, fmt.Errorf("unmarshal response body as JSON: %w", err)
	}

	if token.Error != "" {
		return nil, errors.New(token.Error)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("unexpected response: %s", body)
	}
	return token, nil
}
