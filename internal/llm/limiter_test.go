package llm

import (
	"errors"
	"testing"
	"time"
)

func newTestLimiter(rpm, rpd int, now *time.Time) *Limiter {
	l := NewLimiter(rpm, rpd)
	l.now = func() time.Time { return *now }
	return l
}

func TestLimiterSlidingWindow(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	l := newTestLimiter(2, 100, &now)

	for i := 0; i < 2; i++ {
		if ok, _, err := l.TryAcquire(); !ok || err != nil {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	ok, wait, err := l.TryAcquire()
	if ok || err != nil || wait != time.Minute {
		t.Fatalf("third request should wait 1m, got ok=%v wait=%v err=%v", ok, wait, err)
	}

	now = now.Add(61 * time.Second)
	if ok, _, _ := l.TryAcquire(); !ok {
		t.Fatal("request should be allowed after the window passes")
	}
}

func TestLimiterDailyQuota(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	l := newTestLimiter(10, 2, &now)

	l.TryAcquire()
	l.TryAcquire()
	if _, _, err := l.TryAcquire(); !errors.Is(err, ErrDailyQuota) {
		t.Fatalf("expected ErrDailyQuota, got %v", err)
	}

	now = now.Add(48 * time.Hour)
	if ok, _, err := l.TryAcquire(); !ok || err != nil {
		t.Fatal("quota should reset on a new day")
	}
}

func TestLimiterCoolDown(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	l := newTestLimiter(10, 100, &now)

	l.CoolDown(10 * time.Second)
	ok, wait, _ := l.TryAcquire()
	if ok || wait != 10*time.Second {
		t.Fatalf("expected 10s cool-down, got ok=%v wait=%v", ok, wait)
	}
	now = now.Add(11 * time.Second)
	if ok, _, _ := l.TryAcquire(); !ok {
		t.Fatal("should be allowed after cool-down")
	}
}
