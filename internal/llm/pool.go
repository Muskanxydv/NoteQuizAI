package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// PoolConfig configures a Pool of Gemini models.
type PoolConfig struct {
	APIKey         string
	Models         []string
	RPM            int // per-model requests per minute
	RPD            int // per-model requests per day
	RequestTimeout time.Duration
	MaxRetries     int
	BaseURL        string // optional; override only in tests
}

// Pool implements Client by spreading requests across several Gemini models.
// Each model has its own free-tier quota, so together they multiply throughput.
type Pool struct {
	models     []*geminiModel
	next       atomic.Uint64
	maxRetries int
}

var _ Client = (*Pool)(nil)

func NewPool(cfg PoolConfig) (*Pool, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("llm: API key is required")
	}
	if len(cfg.Models) == 0 {
		return nil, errors.New("llm: at least one model is required")
	}
	if cfg.RPM < 1 || cfg.RPD < 1 {
		return nil, errors.New("llm: RPM and RPD must be positive")
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 60 * time.Second
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}

	httpClient := &http.Client{}
	p := &Pool{maxRetries: cfg.MaxRetries}
	for _, name := range cfg.Models {
		p.models = append(p.models, &geminiModel{
			name:    name,
			apiKey:  cfg.APIKey,
			baseURL: cfg.BaseURL,
			http:    httpClient,
			timeout: cfg.RequestTimeout,
			limiter: NewLimiter(cfg.RPM, cfg.RPD),
		})
	}
	return p, nil
}

// Generate sends the prompt to whichever model has a free slot, retrying
// transient failures with backoff. Safe for concurrent use by many goroutines.
func (p *Pool) Generate(ctx context.Context, prompt string) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, backoff(attempt, retryAfterOf(lastErr))); err != nil {
				return "", err
			}
		}

		m, err := p.acquire(ctx)
		if err != nil {
			return "", err
		}

		text, err := m.call(ctx, prompt)
		if err == nil {
			return text, nil
		}
		lastErr = err

		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests {
			cool := apiErr.RetryAfter
			if cool <= 0 {
				cool = 15 * time.Second
			}
			m.limiter.CoolDown(cool) // steer the next attempt to another model
		}
		if ctx.Err() != nil || !isRetryable(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("llm: giving up after %d attempts: %w", p.maxRetries+1, lastErr)
}

// acquire blocks until some model has a free slot, rotating the starting model
// so load spreads evenly. It fails fast if every model's daily quota is spent.
func (p *Pool) acquire(ctx context.Context) (*geminiModel, error) {
	n := len(p.models)
	for {
		start := int((p.next.Add(1) - 1) % uint64(n))
		minWait := time.Duration(-1)
		exhausted := 0

		for i := 0; i < n; i++ {
			m := p.models[(start+i)%n]
			ok, wait, err := m.limiter.TryAcquire()
			if errors.Is(err, ErrDailyQuota) {
				exhausted++
				continue
			}
			if ok {
				return m, nil
			}
			if minWait < 0 || wait < minWait {
				minWait = wait
			}
		}

		if exhausted == n {
			return nil, ErrDailyQuota
		}
		if minWait < 5*time.Millisecond {
			minWait = 5 * time.Millisecond
		}
		if err := sleepCtx(ctx, minWait); err != nil {
			return nil, err
		}
	}
}
