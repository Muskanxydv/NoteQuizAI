package llm

import (
	"errors"
	"sync"
	"time"
	_ "time/tzdata" // embeds the tz database so America/Los_Angeles loads on Windows too
)

// ErrDailyQuota means a model's (or every model's) daily request budget is spent.
var ErrDailyQuota = errors.New("llm: daily request quota exhausted")

// Limiter enforces one model's limits: a rolling per-minute window (RPM),
// a per-day cap (RPD, resetting at midnight Pacific like Google's quota),
// and an optional cool-down after the API tells us to back off.
type Limiter struct {
	mu           sync.Mutex
	rpm, rpd     int
	window       []time.Time // request times within the last minute
	day          string
	dayCount     int
	blockedUntil time.Time
	loc          *time.Location
	now          func() time.Time // replaceable in tests
}

func NewLimiter(rpm, rpd int) *Limiter {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.UTC
	}
	return &Limiter{rpm: rpm, rpd: rpd, loc: loc, now: time.Now}
}

// TryAcquire takes a request slot if one is free right now. It never blocks.
//   - ok == true: slot taken, go ahead.
//   - ok == false, err == nil: try again after wait.
//   - err == ErrDailyQuota: nothing more today.
func (l *Limiter) TryAcquire() (ok bool, wait time.Duration, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if today := now.In(l.loc).Format("2006-01-02"); today != l.day {
		l.day, l.dayCount = today, 0
	}
	if l.dayCount >= l.rpd {
		return false, 0, ErrDailyQuota
	}
	if now.Before(l.blockedUntil) {
		return false, l.blockedUntil.Sub(now), nil
	}

	cutoff := now.Add(-time.Minute)
	i := 0
	for i < len(l.window) && !l.window[i].After(cutoff) {
		i++
	}
	l.window = l.window[i:]

	if len(l.window) >= l.rpm {
		return false, l.window[0].Add(time.Minute).Sub(now), nil
	}
	l.window = append(l.window, now)
	l.dayCount++
	return true, 0, nil
}

// CoolDown blocks this model for d (used after a 429 from the API).
func (l *Limiter) CoolDown(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if until := l.now().Add(d); until.After(l.blockedUntil) {
		l.blockedUntil = until
	}
}
