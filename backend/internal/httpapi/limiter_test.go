package httpapi

import (
	"testing"
	"time"
)

func TestAttemptTrackerAllow(t *testing.T) {
	tracker := newAttemptTracker()
	now := time.Now()
	window := 10 * time.Minute

	if !tracker.Allow("alpha", 2, window, now) {
		t.Fatal("expected first allow to pass")
	}
	if !tracker.Allow("alpha", 2, window, now) {
		t.Fatal("expected second allow to pass")
	}
	if tracker.Allow("alpha", 2, window, now) {
		t.Fatal("expected third allow to fail")
	}
}

func TestAttemptTrackerAllowResetsAfterWindow(t *testing.T) {
	tracker := newAttemptTracker()
	window := 5 * time.Minute
	now := time.Now()

	if !tracker.Allow("beta", 1, window, now) {
		t.Fatal("expected allow to pass")
	}
	if tracker.Allow("beta", 1, window, now) {
		t.Fatal("expected second allow to fail")
	}

	later := now.Add(window + time.Second)
	if !tracker.Allow("beta", 1, window, later) {
		t.Fatal("expected allow to pass after window reset")
	}
}

func TestAttemptTrackerRegisterFailureLocks(t *testing.T) {
	tracker := newAttemptTracker()
	now := time.Now()
	window := 15 * time.Minute
	lock := 10 * time.Minute

	if tracker.RegisterFailure("gamma", 3, window, lock, now) {
		t.Fatal("expected not locked on first failure")
	}
	if tracker.RegisterFailure("gamma", 3, window, lock, now) {
		t.Fatal("expected not locked on second failure")
	}
	if !tracker.RegisterFailure("gamma", 3, window, lock, now) {
		t.Fatal("expected locked on third failure")
	}
	if !tracker.IsLocked("gamma", now.Add(time.Minute)) {
		t.Fatal("expected lock to be active")
	}
	if tracker.IsLocked("gamma", now.Add(lock+time.Second)) {
		t.Fatal("expected lock to expire")
	}
}

func TestAttemptTrackerReset(t *testing.T) {
	tracker := newAttemptTracker()
	now := time.Now()

	if !tracker.Allow("delta", 1, time.Minute, now) {
		t.Fatal("expected allow to pass")
	}
	tracker.Reset("delta")
	if !tracker.Allow("delta", 1, time.Minute, now) {
		t.Fatal("expected allow to pass after reset")
	}
}
