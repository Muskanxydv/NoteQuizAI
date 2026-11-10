package handler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"notequizai/internal/llm"
	"notequizai/internal/models"
	"notequizai/internal/service"
	"notequizai/internal/storage"
)

// fakeService is a stand-in used only by these tests.
type fakeService struct {
	create func(ctx context.Context, req models.GenerateRequest) (*models.Assessment, error)
	get    func(ctx context.Context, id string) (*models.Assessment, error)
}

func (f fakeService) Create(ctx context.Context, req models.GenerateRequest) (*models.Assessment, error) {
	return f.create(ctx, req)
}

func (f fakeService) Get(ctx context.Context, id string) (*models.Assessment, error) {
	return f.get(ctx, id)
}

func newMux(svc AssessmentService) *http.ServeMux {
	mux := http.NewServeMux()
	NewAssessmentHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second).Register(mux)
	return mux
}

func do(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const validBody = `{"notes":"hello","questions":[{"type":"1-mark","count":2}]}`

func TestCreateReturns201(t *testing.T) {
	mux := newMux(fakeService{create: func(context.Context, models.GenerateRequest) (*models.Assessment, error) {
		return &models.Assessment{ID: "abc"}, nil
	}})
	rec := do(mux, http.MethodPost, "/api/v1/assessments", validBody)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"abc"`) {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestCreateRejectsBadJSON(t *testing.T) {
	mux := newMux(fakeService{})
	for _, body := range []string{`not json`, `{"nope":1}`} {
		if rec := do(mux, http.MethodPost, "/api/v1/assessments", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("body %q: expected 400, got %d", body, rec.Code)
		}
	}
}

func TestCreateErrorMapping(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{fmt.Errorf("%w: notes must not be empty", service.ErrInvalidRequest), http.StatusBadRequest},
		{fmt.Errorf("all tasks failed: %w", llm.ErrDailyQuota), http.StatusTooManyRequests},
		{llm.ErrBlocked, http.StatusUnprocessableEntity},
		{context.DeadlineExceeded, http.StatusGatewayTimeout},
		{&llm.APIError{StatusCode: 503, Message: "down"}, http.StatusBadGateway},
		{fmt.Errorf("something odd"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		err := tc.err
		mux := newMux(fakeService{create: func(context.Context, models.GenerateRequest) (*models.Assessment, error) {
			return nil, err
		}})
		if rec := do(mux, http.MethodPost, "/api/v1/assessments", validBody); rec.Code != tc.want {
			t.Errorf("error %v: expected %d, got %d", tc.err, tc.want, rec.Code)
		}
	}
}

func TestGetAssessment(t *testing.T) {
	mux := newMux(fakeService{get: func(_ context.Context, id string) (*models.Assessment, error) {
		if id == "abc" {
			return &models.Assessment{ID: "abc"}, nil
		}
		return nil, storage.ErrNotFound
	}})
	if rec := do(mux, http.MethodGet, "/api/v1/assessments/abc", ""); rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec := do(mux, http.MethodGet, "/api/v1/assessments/zzz", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestWrongMethodIs405(t *testing.T) {
	mux := newMux(fakeService{})
	if rec := do(mux, http.MethodDelete, "/api/v1/assessments", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
