package agent

import (
	"encoding/json"
	"sync"
)

// store is an in-memory token cache keyed by client ID.
// Tokens are stored as opaque JSON so that the agent does not depend on the
// concrete access token type defined in the ghtkn SDK. This keeps the client-side
// implementation free to evolve independently.
//
// The cache lives only for the lifetime of the agent process. Encrypted on-disk
// persistence is planned for a later change.
type store struct {
	mu     sync.Mutex
	tokens map[string]json.RawMessage
}

// newStore creates an empty in-memory token store.
func newStore() *store {
	return &store{
		tokens: map[string]json.RawMessage{},
	}
}

// Get returns the cached token for the given client ID.
// The second return value is false when no token is cached for the client ID.
func (s *store) Get(clientID string) (json.RawMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[clientID]
	return token, ok
}

// Set stores a token for the given client ID, replacing any existing entry.
func (s *store) Set(clientID string, token json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[clientID] = token
}

// Len returns the number of cached tokens.
func (s *store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tokens)
}
