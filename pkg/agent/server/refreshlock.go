package server

import (
	"context"
	"crypto/sha256"
	"sync/atomic"
)

// refreshLock is the per-client refresh lock (see lockRefresh) and the state guarded by
// it. sem is a one-slot semaphore: a channel rather than a sync.Mutex so the wait can be
// abandoned on ctx cancellation, as the holder may be blocked on a GitHub request for up
// to githubHTTPTimeout.
type refreshLock struct {
	sem chan struct{}
	// failures counts the refreshes that failed under this lock. It is only incremented
	// while holding sem, but it is atomic because lockRefresh reads it before waiting on
	// sem, to tell whether a refresh failed while the caller waited.
	failures atomic.Uint64
	// failedDigest is the SHA-256 of the refresh token the last failed refresh used, and
	// failureWarning that refresh's warning. A digest is kept rather than the token so no
	// plaintext refresh token outlives the request. Read and written only while holding
	// sem.
	failedDigest   [sha256.Size]byte
	failureWarning string
}

// lockRefresh acquires the per-client refresh lock for clientID, waiting while another
// GET for the same client is refreshing. It returns the held lock, to be released with
// unlock, or ctx's error if ctx is done first (the agent is shutting down).
//
// The second result reports whether a refresh for clientID failed while this caller was
// waiting. Such a caller is queued behind the failed attempt; if the stored refresh token
// is still the one that failed (see failedWith), it should fall back with that refresh's
// warning instead of retrying it (see refreshAccessToken). A refresh that failed before
// this caller started waiting does not count, so a later GET tries the refresh token
// again: the earlier failure may have been transient.
func (s *Server) lockRefresh(ctx context.Context, clientID string) (*refreshLock, bool, error) {
	s.refreshLocksMu.Lock()
	lock, ok := s.refreshLocks[clientID]
	if !ok {
		lock = &refreshLock{sem: make(chan struct{}, 1)}
		s.refreshLocks[clientID] = lock
	}
	s.refreshLocksMu.Unlock()
	failures := lock.failures.Load()
	select {
	case lock.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, false, ctx.Err() //nolint:wrapcheck
	}
	return lock, lock.failures.Load() != failures, nil
}

// unlock releases the lock acquired by lockRefresh.
func (l *refreshLock) unlock() {
	<-l.sem
}

// markFailed records that a refresh with refreshToken failed with warning (see
// lockRefresh). The caller must hold the lock.
func (l *refreshLock) markFailed(refreshToken, warning string) {
	l.failedDigest = sha256.Sum256([]byte(refreshToken))
	l.failureWarning = warning
	l.failures.Add(1)
}

// failedWith reports whether refreshToken is the one the last failed refresh used. A
// refresh token that was rotated by a later successful refresh, or replaced by a device
// flow, does not match. The caller must hold the lock.
func (l *refreshLock) failedWith(refreshToken string) bool {
	return refreshToken != "" && sha256.Sum256([]byte(refreshToken)) == l.failedDigest
}
