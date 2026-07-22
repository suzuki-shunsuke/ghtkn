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
// Tokens are encrypted with dataKey (AES-256-GCM) and persisted under dir as one file
// per client ID. The store holds no field for the plaintext, so a decrypted token only
// lives for the duration of the request that reads it: each Get reads and decrypts the
// file anew, keeping recognizable access/refresh tokens (which carry scannable
// "ghu_"/"ghr_" prefixes) out of a memory dump. Tokens are opaque JSON so the agent does
// not depend on the concrete access token type defined in the ghtkn SDK.
type Store struct {
	// mu serializes access to the token files so concurrent requests for the same client
	// ID cannot interleave a read with a write or a delete.
	mu      sync.Mutex
	dataKey []byte
	dir     string
}

// New creates a token store that persists encrypted tokens under dir,
// encrypting them with dataKey. dir must not be empty.
func New(dataKey []byte, dir string) *Store {
	return &Store{
		dataKey: dataKey,
		dir:     dir,
	}
}

// Get returns the token for clientID. The bool result is false when no token is stored
// for the client ID. It reads and decrypts the token file on every call and does NOT
// cache the plaintext, so the decrypted token is not retained in memory between
// requests; the caller should scrub the returned bytes once done with them.
func (s *Store) Get(clientID string) (json.RawMessage, bool, error) {
	if !validClientID(clientID) {
		return nil, false, ErrInvalidClientID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

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
	return json.RawMessage(plaintext), true, nil
}

// Set stores a token for clientID: it encrypts the token and writes it atomically to
// disk without retaining the plaintext in memory.
func (s *Store) Set(clientID string, token json.RawMessage) error {
	if !validClientID(clientID) {
		return ErrInvalidClientID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	blob, err := crypt.Seal(s.dataKey, token)
	if err != nil {
		return fmt.Errorf("encrypt the token: %w", err)
	}
	if err := crypt.AtomicWrite(filepath.Join(s.dir, clientID), blob); err != nil {
		return fmt.Errorf("write the token file: %w", err)
	}
	return nil
}

// Delete removes the token stored for clientID. Deleting a client ID with no stored
// token is a no-op (a missing file is not an error), so callers can delete
// unconditionally.
func (s *Store) Delete(clientID string) error {
	if !validClientID(clientID) {
		return ErrInvalidClientID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(filepath.Join(s.dir, clientID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove the token file: %w", err)
	}
	return nil
}

// Len returns the number of stored tokens by counting the valid token files on disk
// (ignoring temporary files and invalid names). A read error yields 0 so that STATUS
// stays infallible.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.diskClientIDs())
}

// ClientIDs returns the client IDs of all stored tokens, listing the valid token files
// on disk (ignoring temporary files and invalid names). It lets callers iterate every
// stored token, e.g. to sweep expired ones or strip refresh tokens.
func (s *Store) ClientIDs() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read the token directory: %w", err)
	}
	return diskClientIDsFromEntries(entries), nil
}

// diskClientIDs lists the client IDs of the token files under s.dir. It returns nil on
// a read error so that Len stays infallible. The caller must hold s.mu.
func (s *Store) diskClientIDs() []string {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil
	}
	return diskClientIDsFromEntries(entries)
}

// diskClientIDsFromEntries filters directory entries down to valid token file names,
// skipping subdirectories, temporary files, and names that are not valid client IDs.
func diskClientIDsFromEntries(entries []os.DirEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".ghtkn-tmp-") {
			continue
		}
		if validClientID(e.Name()) {
			ids = append(ids, e.Name())
		}
	}
	return ids
}
