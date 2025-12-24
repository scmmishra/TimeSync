package httpapi

import (
	"sync"
	"time"
)

type attemptTracker struct {
	mu    sync.Mutex
	state map[string]*attemptState
}

type attemptState struct {
	count     int
	resetAt   time.Time
	lockUntil time.Time
}

func newAttemptTracker() *attemptTracker {
	return &attemptTracker{
		state: make(map[string]*attemptState),
	}
}

func (t *attemptTracker) Allow(key string, max int, window time.Duration, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	st := t.state[key]
	if st == nil || now.After(st.resetAt) {
		t.state[key] = &attemptState{
			count:   1,
			resetAt: now.Add(window),
		}
		return true
	}

	if st.count >= max {
		return false
	}
	st.count++
	return true
}

func (t *attemptTracker) RegisterFailure(key string, max int, window, lock time.Duration, now time.Time) (locked bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	st := t.state[key]
	if st == nil || now.After(st.resetAt) {
		st = &attemptState{
			count:   0,
			resetAt: now.Add(window),
		}
		t.state[key] = st
	}

	if now.Before(st.lockUntil) {
		return true
	}

	st.count++
	if st.count >= max {
		st.lockUntil = now.Add(lock)
		return true
	}
	return false
}

func (t *attemptTracker) IsLocked(key string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	st := t.state[key]
	if st == nil {
		return false
	}
	return now.Before(st.lockUntil)
}

func (t *attemptTracker) Reset(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.state, key)
}
