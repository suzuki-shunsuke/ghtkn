package keyring

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/zalando/go-keyring"
)

type Keyring struct {
	keyService string
}

func New(keyService string) *Keyring {
	return &Keyring{
		keyService: keyService,
	}
}

const dateFormat = time.RFC3339

func ParseDate(s string) (time.Time, error) {
	t, err := time.Parse(dateFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse a date string: %w", err)
	}
	return t, nil
}

func FormatDate(t time.Time) string {
	return t.Format(dateFormat)
}

type AccessToken struct {
	App            string `json:"app"`
	AccessToken    string `json:"access_token"`
	ExpirationDate string `json:"expiration_date"`
	// ClientID string `json:"client_id"`
}

func (kr *Keyring) Get(key string) (*AccessToken, error) {
	s, err := keyring.Get(kr.keyService, key)
	if err != nil {
		return nil, fmt.Errorf("get a GitHub Access token in keyring: %w", err)
	}
	token := &AccessToken{}
	if err := json.Unmarshal([]byte(s), token); err != nil {
		return nil, fmt.Errorf("unmarshal the token as JSON: %w", err)
	}
	return token, nil
}

func (kr *Keyring) Set(key string, token *AccessToken) error {
	s, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal the token as JSON: %w", err)
	}
	if err := keyring.Set(kr.keyService, key, string(s)); err != nil {
		return fmt.Errorf("set a GitHub Access token in keyring: %w", err)
	}
	return nil
}

func (kr *Keyring) Remove(logger *slog.Logger, key string) error {
	if err := keyring.Delete(kr.keyService, key); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			slogerr.WithError(logger, err).Warn("tried to remove a GitHub Access token from keyring, but the key wasn't found")
			return nil
		}
		return fmt.Errorf("remove a GitHub Access token from keyring: %w", err)
	}
	return nil
}
