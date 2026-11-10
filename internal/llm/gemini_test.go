package llm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const okBody = `{"candidates":[{"content":{"parts":[{"text":"{\"ok\":true}"}]},"finishReason":"STOP"}]}`

func newTestPool(t *testing.T, url string, models []string, rpd int) *Pool {
	t.Helper()
	p, err := NewPool(PoolConfig{
		APIKey:         "test-key",
		Models:         models,
		RPM:            10,
		RPD:            rpd,
		RequestTimeout: 5 * time.Second,
		MaxRetries:     2,
		BaseURL:        url,
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGenerateSuccess(t *testing.T) {
	var gotKey, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-goog-api-key")
		gotPath = r.URL.Path
		w.Write([]byte(okBody))
	}))
	defer srv.Close()

	p := newTestPool(t, srv.URL, []string{"model-a"}, 10)
	out, err := p.Generate(context.Background(), "hi")
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"ok":true}` {
		t.Fatalf("unexpected output %q", out)
	}
	if gotKey != "test-key" {
		t.Fatalf("API key header missing, got %q", gotKey)
	}
	if !strings.HasSuffix(gotPath, "/models/model-a:generateContent") {
		t.Fatalf("unexpected path %q", gotPath)
	}
}

func TestRetryOn429UsesOtherModel(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"quota"}}`))
			return
		}
		w.Write([]byte(okBody))
	}))
	defer srv.Close()

	p := newTestPool(t, srv.URL, []string{"model-a", "model-b"}, 10)
	if _, err := p.Generate(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", calls.Load())
	}
}

func TestBlockedResponseIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"promptFeedback":{"blockReason":"SAFETY"}}`))
	}))
	defer srv.Close()

	p := newTestPool(t, srv.URL, []string{"model-a"}, 10)
	_, err := p.Generate(context.Background(), "hi")
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected ErrBlocked, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("blocked responses must not be retried, got %d calls", calls.Load())
	}
}

func TestDailyQuotaExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(okBody))
	}))
	defer srv.Close()

	p := newTestPool(t, srv.URL, []string{"model-a", "model-b"}, 1)
	for i := 0; i < 2; i++ {
		if _, err := p.Generate(context.Background(), "hi"); err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
	}
	if _, err := p.Generate(context.Background(), "hi"); !errors.Is(err, ErrDailyQuota) {
		t.Fatalf("expected ErrDailyQuota, got %v", err)
	}
}
