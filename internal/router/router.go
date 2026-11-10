// Package router wires routes and middleware together.
package router

import (
	"log/slog"
	"net/http"

	"notequizai/internal/handler"
	"notequizai/internal/middleware"
)

func New(h *handler.AssessmentHandler, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)
	h.Register(mux)

	// First listed = outermost: the request ID exists before logging and recovery run.
	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Logging(log),
		middleware.Recovery(log),
	)
}
