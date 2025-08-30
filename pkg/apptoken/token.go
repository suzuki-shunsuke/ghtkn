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

type Client struct {
	httpClient *http.Client
	now        func() time.Time
	stderr     io.Writer
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{
		httpClient: httpClient,
		now:        time.Now,
		stderr:     os.Stderr,
	}
}

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`

	Error string `json:"error"`
}

type AccessToken struct {
	App            string `json:"app"`
	AccessToken    string `json:"access_token"`
	ExpirationDate string `json:"expiration_date"`
}

type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (c *Client) Create(ctx context.Context, logger *slog.Logger, clientID string) (*AccessToken, error) {
	if clientID == "" {
		return nil, errors.New("client id is required")
	}
	deviceCode, err := c.getDeviceCode(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("get device code: %w", err)
	}

	fmt.Fprintf(c.stderr, "Please visit: %s\n", deviceCode.VerificationURI)
	fmt.Fprintf(c.stderr, "And enter code: %s\n", deviceCode.UserCode)
	fmt.Fprintf(c.stderr, "Expiration date: %s\n", c.now().Add(time.Duration(deviceCode.ExpiresIn)*time.Second).Format(time.RFC3339))
	if err := openBrowser(ctx, deviceCode.VerificationURI); err != nil {
		if !errors.Is(err, errNoCommandFound) {
			slogerr.WithError(logger, err).Warn("failed to open the browser")
		}
	}

	token, err := c.pollForAccessToken(ctx, clientID, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	now := c.now()

	return &AccessToken{
		AccessToken:    token.AccessToken,
		ExpirationDate: keyring.FormatDate(now.Add(time.Duration(token.ExpiresIn) * time.Second)),
	}, nil
}

func (c *Client) getDeviceCode(ctx context.Context, clientID string) (*DeviceCodeResponse, error) {
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

	resp, err := c.httpClient.Do(req)
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

const additionalInterval = 5 * time.Second

func (c *Client) pollForAccessToken(ctx context.Context, clientID string, deviceCode *DeviceCodeResponse) (*AccessTokenResponse, error) {
	interval := time.Duration(deviceCode.Interval) * time.Second
	if interval < additionalInterval {
		interval = additionalInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deadline := time.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context was cancelled: %w", ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
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

	resp, err := c.httpClient.Do(req)
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
