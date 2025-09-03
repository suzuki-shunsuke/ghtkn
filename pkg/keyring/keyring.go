// Package keyring provides secure storage for GitHub access tokens.
// It wraps the zalando/go-keyring library to store and retrieve tokens from the system keychain.
package keyring

import (
	"encoding/json"
	"fmt"
	"time"
)

// Keyring manages access tokens in the system keychain.
// It provides methods to get, set, and remove tokens securely.
type Keyring struct {
	input *Input
}

type Input struct {
	KeyService string
	API        API
}

// New creates a new Keyring instance with the specified service name.
// The keyService parameter is used as the service identifier in the system keychain.
func New(input *Input) *Keyring {
	return &Keyring{
		input: input,
	}
}

type API interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
}

func NewInput(key string) *Input {
	return &Input{
		KeyService: key,
		API:        NewAPI(),
	}
}

func NewMockInput(key string, secrets map[string]*AccessToken) *Input {
	return &Input{
		KeyService: key,
		API:        NewMockAPI(secrets),
	}
}

// dateFormat defines the standard format for date strings in the keyring.
const dateFormat = time.RFC3339

// ParseDate parses a date string in RFC3339 format.
// It returns a time.Time value or an error if the string cannot be parsed.
func ParseDate(s string) (time.Time, error) {
	t, err := time.Parse(dateFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse a date string: %w", err)
	}
	return t, nil
}

// FormatDate formats a time value as an RFC3339 string.
// This is the standard format used for expiration dates in the keyring.
func FormatDate(t time.Time) string {
	return t.Format(dateFormat)
}

// AccessToken represents a GitHub App access token stored in the keyring.
// It contains the token value, expiration information, and user login details.
type AccessToken struct {
	App            string `json:"app"`             // The GitHub App name for display purposes
	AccessToken    string `json:"access_token"`    // The OAuth access token for GitHub API authentication
	ExpirationDate string `json:"expiration_date"` // RFC3339 formatted expiration timestamp
	Login          string `json:"login"`           // GitHub username for Git Credential Helper authentication
	// ClientID string `json:"client_id"`
}

// Get retrieves an access token from the keyring.
// The key parameter identifies the token to retrieve.
// Returns the token or an error if the token cannot be found or unmarshaled.
func (kr *Keyring) Get(key string) (*AccessToken, error) {
	s, err := kr.input.API.Get(kr.input.KeyService, key)
	if err != nil {
		return nil, fmt.Errorf("get a GitHub Access token in keyring: %w", err)
	}
	token := &AccessToken{}
	if err := json.Unmarshal([]byte(s), token); err != nil {
		return nil, fmt.Errorf("unmarshal the token as JSON: %w", err)
	}
	return token, nil
}

// Set stores an access token in the keyring.
// The key parameter identifies where to store the token.
// Returns an error if the token cannot be marshaled or stored.
func (kr *Keyring) Set(key string, token *AccessToken) error {
	s, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal the token as JSON: %w", err)
	}
	if err := kr.input.API.Set(kr.input.KeyService, key, string(s)); err != nil {
		return fmt.Errorf("set a GitHub Access token in keyring: %w", err)
	}
	return nil
}
