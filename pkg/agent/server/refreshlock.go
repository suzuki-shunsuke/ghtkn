package server

import (
	"context"
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
	// failureWarning is the warning of the last failed refresh, cleared by a later
	// successful one. Read and written only while holding sem.
	failureWarning string
}

// lockRefresh acquires the per-client refresh lock for clientID, waiting while another
// GET for the same client is refreshing. It returns the held lock, to be released with
// unlock, or ctx's error if ctx is done first (the agent is shutting down).
//
// The second result is non-empty when a refresh for clientID failed while this caller
// was waiting, and no refresh succeeded after it: it is that refresh's warning. Such a caller is queued behind the failed
// attempt and would retry the same refresh token, so it should fall back with the same
// warning instead (see refreshAccessToken). A refresh that failed before this caller
// started waiting does not count, so a later GET tries the refresh token again: the
// earlier failure may have been transient.
func (s *Server) lockRefresh(ctx context.Context, clientID string) (*refreshLock, string, error) {
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
		return nil, "", ctx.Err() //nolint:wrapcheck
	}
	if lock.failures.Load() != failures {
		return lock, lock.failureWarning, nil
	}
	return lock, "", nil
}

// markSucceeded records that a refresh succeeded, so the callers waiting on the lock
// across an earlier failure try the new refresh token instead of falling back with the
// stale warning. The caller must hold the lock.
func (l *refreshLock) markSucceeded() {
	l.failureWarning = ""
}

// unlock releases the lock acquired by lockRefresh.
func (l *refreshLock) unlock() {
	<-l.sem
}

// markFailed records that a refresh failed with warning, so the callers waiting on the
// lock fall back with it (see lockRefresh). The caller must hold the lock.
func (l *refreshLock) markFailed(warning string) {
	l.failureWarning = warning
	l.failures.Add(1)
}
