package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// clientIDPattern restricts client IDs to characters that are safe to use directly
// as a file name. GitHub App client IDs (e.g. "Iv1.<hex>", "Iv23<...>") match it.
var clientIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// errInvalidClientID is returned when a client ID is unsafe to use as a file name.
var errInvalidClientID = errors.New("invalid client id")

// validClientID reports whether id is safe to use as a token file name.
// It rejects empty strings, "." and "..", and anything outside clientIDPattern,
// which prevents path traversal.
func validClientID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	return clientIDPattern.MatchString(id)
}

// store caches access tokens keyed by client ID.
//
// In memory-only mode (dataKey == nil, dir == "") tokens live only for the lifetime
// of the process. In disk mode tokens are encrypted with dataKey (AES-256-GCM) and
// persisted under dir as one file per client ID; the in-memory map is a lazily
// populated cache. Tokens are opaque JSON so the agent does not depend on the
// concrete access token type defined in the ghtkn SDK.
type store struct {
	mu      sync.Mutex
	tokens  map[string]json.RawMessage
	dataKey []byte // nil in memory-only mode
	dir     string // "" in memory-only mode
}

// newStore creates an empty in-memory (non-persistent) token store.
func newStore() *store {
	return &store{
		tokens: map[string]json.RawMessage{},
	}
}

// newDiskStore creates a token store that persists encrypted tokens under dir,
// encrypting them with dataKey.
func newDiskStore(dataKey []byte, dir string) *store {
	return &store{
		tokens:  map[string]json.RawMessage{},
		dataKey: dataKey,
		dir:     dir,
	}
}

// Get returns the cached token for clientID. The bool result is false when no token
// is cached for the client ID. In disk mode a miss falls through to reading and
// decrypting the token file, caching it on success.
func (s *store) Get(clientID string) (json.RawMessage, bool, error) {
	if !validClientID(clientID) {
		return nil, false, errInvalidClientID
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
	plaintext, err := open(s.dataKey, blob)
	if err != nil {
		return nil, false, fmt.Errorf("decrypt the token file: %w", err)
	}
	token := json.RawMessage(plaintext)
	s.tokens[clientID] = token
	return token, true, nil
}

// Set stores a token for clientID. In disk mode it encrypts the token and writes it
// atomically before updating the in-memory cache.
func (s *store) Set(clientID string, token json.RawMessage) error {
	if !validClientID(clientID) {
		return errInvalidClientID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dir != "" {
		blob, err := seal(s.dataKey, token)
		if err != nil {
			return fmt.Errorf("encrypt the token: %w", err)
		}
		if err := atomicWrite(filepath.Join(s.dir, clientID), blob); err != nil {
			return fmt.Errorf("write the token file: %w", err)
		}
	}
	s.tokens[clientID] = token
	return nil
}

// Len returns the number of cached tokens. In disk mode it counts the valid token
// files on disk (ignoring temporary files and invalid names), since lazy loading
// means the in-memory map only reflects tokens touched since start. A read error
// yields 0 so that STATUS stays infallible.
func (s *store) Len() int {
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
