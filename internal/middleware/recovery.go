package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery turns a panic in a handler into a 500 response instead of
// crashing the whole server.
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					log.Error("panic recovered",
						"panic", p,
						"stack", string(debug.Stack()),
						"request_id", RequestIDFrom(r.Context()),
					)
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":{"code":"internal_error","message":"Something went wrong."}}` + "\n"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
