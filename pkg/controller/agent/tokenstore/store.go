// Package tokenstore caches GitHub App access tokens for the agent, encrypted at
// rest with AES-256-GCM (via the crypt package) under the data key produced by the
// keyfile package. It also resolves the directory the token files live in.
package tokenstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/crypt"
)

// clientIDPattern restricts client IDs to characters that are safe to use directly
// as a file name. GitHub App client IDs (e.g. "Iv1.<hex>", "Iv23<...>") match it.
var clientIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ErrInvalidClientID is returned when a client ID is unsafe to use as a file name.
var ErrInvalidClientID = errors.New("invalid client id")

// ErrDecryptToken is returned (wrapped) when a persisted token file exists but
// can't be decrypted with the current data key, e.g. after the agent key was
// rotated. Callers can detect it with errors.Is to treat the stale token as a
// cache miss rather than a hard failure.
var ErrDecryptToken = errors.New("decrypt the token file")

// validClientID reports whether id is safe to use as a token file name.
// It rejects empty strings, "." and "..", and anything outside clientIDPattern,
// which prevents path traversal.
func validClientID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	return clientIDPattern.MatchString(id)
}

// Store caches access tokens keyed by client ID.
//
// In memory-only mode (dataKey == nil, dir == "") tokens live only for the lifetime
// of the process. In disk mode tokens are encrypted with dataKey (AES-256-GCM) and
// persisted under dir as one file per client ID; the in-memory map is a lazily
// populated cache. Tokens are opaque JSON so the agent does not depend on the
// concrete access token type defined in the ghtkn SDK.
type Store struct {
	mu      sync.Mutex
	tokens  map[string]json.RawMessage
	dataKey []byte // nil in memory-only mode
	dir     string // "" in memory-only mode
}

// New creates a token store that persists encrypted tokens under dir,
// encrypting them with dataKey.
func New(dataKey []byte, dir string) *Store {
	return &Store{
		tokens:  map[string]json.RawMessage{},
		dataKey: dataKey,
		dir:     dir,
	}
}

// Get returns the cached token for clientID. The bool result is false when no token
// is cached for the client ID. In disk mode a miss falls through to reading and
// decrypting the token file, caching it on success.
func (s *Store) Get(clientID string) (json.RawMessage, bool, error) {
	if !validClientID(clientID) {
		return nil, false, ErrInvalidClientID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if token, ok := s.tokens[clientID]; ok {
		return token, true, nil
	}
	if s.dir == "" {
		return nil, false, nil
	}

	blob, err := os.ReadFile(filepath.Join(s.dir, clientID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read the token file: %w", err)
	}
	plaintext, err := crypt.Open(s.dataKey, blob)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %w", ErrDecryptToken, err)
	}
	token := json.RawMessage(plaintext)
	s.tokens[clientID] = token
	return token, true, nil
}

// Set stores a token for clientID. In disk mode it encrypts the token and writes it
// atomically before updating the in-memory cache.
func (s *Store) Set(clientID string, token json.RawMessage) error {
	if !validClientID(clientID) {
		return ErrInvalidClientID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dir != "" {
		blob, err := crypt.Seal(s.dataKey, token)
		if err != nil {
			return fmt.Errorf("encrypt the token: %w", err)
		}
		if err := crypt.AtomicWrite(filepath.Join(s.dir, clientID), blob); err != nil {
			return fmt.Errorf("write the token file: %w", err)
		}
	}
	s.tokens[clientID] = token
	return nil
}

// Delete removes the token cached for clientID from memory and, in disk mode, from
// disk. Deleting a client ID with no cached token is a no-op (a missing file is not
// an error), so callers can delete unconditionally.
func (s *Store) Delete(clientID string) error {
	if !validClientID(clientID) {
		return ErrInvalidClientID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dir != "" {
		if err := os.Remove(filepath.Join(s.dir, clientID)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove the token file: %w", err)
		}
	}
	delete(s.tokens, clientID)
	return nil
}

// Len returns the number of cached tokens. In disk mode it counts the valid token
// files on disk (ignoring temporary files and invalid names), since lazy loading
// means the in-memory map only reflects tokens touched since start. A read error
// yields 0 so that STATUS stays infallible.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dir == "" {
		return len(s.tokens)
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".ghtkn-tmp-") {
			continue
		}
		if validClientID(e.Name()) {
			n++
		}
	}
	return n
}
