// Package middleware contains HTTP middleware.
package middleware

import "net/http"

// Chain wraps h so that mws[0] is the outermost layer.
func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
