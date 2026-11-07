package server

import (
	"context"
	"crypto/sha256"
	"time"
)

// refreshFailureWindow is how long a failed refresh token is remembered (see
// refreshLock.recentlyFailed). It only needs to cover the GETs that queued up behind the
// failed attempt, which run right after it. Keeping it short means a refresh token that
// failed for a transient reason (a GitHub 5xx or a timeout) is tried again by a later GET
// rather than being given up until a device flow replaces it.
const refreshFailureWindow = time.Minute

// refreshLock is the per-client refresh lock (see lockRefresh) and the state guarded by
// it. sem is a one-slot semaphore: a channel rather than a sync.Mutex so the wait can be
// abandoned on ctx cancellation, as the holder may be blocked on a GitHub request for up
// to githubHTTPTimeout. The other fields are read and written only while holding sem.
type refreshLock struct {
	sem chan struct{}
	// failedDigest is the SHA-256 of the last refresh token that failed to refresh, and
	// failedAt when it failed. A digest is kept rather than the token itself so no
	// plaintext refresh token outlives the request.
	failedDigest [sha256.Size]byte
	failedAt     time.Time
}

// lockRefresh acquires the per-client refresh lock for clientID, waiting while another
// GET for the same client is refreshing. It returns the held lock, to be released with
// unlock, or ctx's error if ctx is done first (the agent is shutting down).
func (s *Server) lockRefresh(ctx context.Context, clientID string) (*refreshLock, error) {
	s.refreshLocksMu.Lock()
	lock, ok := s.refreshLocks[clientID]
	if !ok {
		lock = &refreshLock{sem: make(chan struct{}, 1)}
		s.refreshLocks[clientID] = lock
	}
	s.refreshLocksMu.Unlock()
	select {
	case lock.sem <- struct{}{}:
		return lock, nil
	case <-ctx.Done():
		return nil, ctx.Err() //nolint:wrapcheck
	}
}

// unlock releases the lock acquired by lockRefresh.
func (l *refreshLock) unlock() {
	<-l.sem
}

// markFailed records that refreshToken failed to refresh. The caller must hold the lock.
func (l *refreshLock) markFailed(refreshToken string) {
	l.failedDigest = sha256.Sum256([]byte(refreshToken))
	l.failedAt = time.Now()
}

// recentlyFailed reports whether refreshToken is the one that failed to refresh within
// refreshFailureWindow. The caller must hold the lock.
func (l *refreshLock) recentlyFailed(refreshToken string) bool {
	if l.failedAt.IsZero() || time.Since(l.failedAt) >= refreshFailureWindow {
		return false
	}
	return sha256.Sum256([]byte(refreshToken)) == l.failedDigest
}
