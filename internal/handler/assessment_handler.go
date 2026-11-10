package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"notequizai/internal/llm"
	"notequizai/internal/middleware"
	"notequizai/internal/models"
	"notequizai/internal/service"
	"notequizai/internal/storage"
	"notequizai/internal/validator"
)

const maxBodyBytes = 1 << 20 // 1 MB

// AssessmentService is what the handler needs from the service layer.
type AssessmentService interface {
	Create(ctx context.Context, req models.GenerateRequest) (*models.Assessment, error)
	Get(ctx context.Context, id string) (*models.Assessment, error)
}

type AssessmentHandler struct {
	svc     AssessmentService
	log     *slog.Logger
	timeout time.Duration
}

func NewAssessmentHandler(svc AssessmentService, log *slog.Logger, timeout time.Duration) *AssessmentHandler {
	return &AssessmentHandler{svc: svc, log: log, timeout: timeout}
}

// Register adds the assessment routes to mux (Go 1.22 method+path patterns).
func (h *AssessmentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/assessments", h.create)
	mux.HandleFunc("GET /api/v1/assessments/{id}", h.get)
}

func (h *AssessmentHandler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req models.GenerateRequest
	if err := dec.Decode(&req); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body must be under 1 MB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	a, err := h.svc.Create(ctx, req)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *AssessmentHandler) get(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// writeServiceError maps internal errors to HTTP responses. Details of
// unexpected failures go to the log, never to the client.
func (h *AssessmentHandler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	reqID := middleware.RequestIDFrom(r.Context())
	var apiErr *llm.APIError

	switch {
	case errors.Is(err, service.ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, storage.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "assessment not found")
	case errors.Is(err, llm.ErrDailyQuota):
		writeError(w, http.StatusTooManyRequests, "daily_quota_exhausted", "The free AI quota is used up for today. Please try again tomorrow.")
	case errors.Is(err, llm.ErrBlocked):
		writeError(w, http.StatusUnprocessableEntity, "content_blocked", "The AI could not process these notes.")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "timeout", "Generation took too long. Try fewer topics or question types.")
	case errors.Is(err, context.Canceled):
		h.log.Info("client cancelled request", "request_id", reqID)
	case errors.As(err, &apiErr):
		h.log.Error("AI provider error", "error", err, "request_id", reqID)
		writeError(w, http.StatusBadGateway, "ai_provider_error", "The AI provider returned an error. Please try again.")
	case errors.Is(err, validator.ErrInvalid):
		h.log.Error("AI returned invalid output", "error", err, "request_id", reqID)
		writeError(w, http.StatusBadGateway, "invalid_ai_output", "The AI returned an unusable answer. Please try again.")
	default:
		h.log.Error("unexpected error", "error", err, "request_id", reqID)
		writeError(w, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}
