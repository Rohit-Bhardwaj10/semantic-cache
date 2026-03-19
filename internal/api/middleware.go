package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestIDMiddleware injects a unique X-Request-ID into each request context.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			// Generate a simple ID
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			rid = fmt.Sprintf("%x", b)
		}

		ctx := context.WithValue(r.Context(), requestIDKey, rid)
		w.Header().Set("X-Request-ID", rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggerMiddleware logs every incoming request.
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("[%s] %s %s %v\n", r.Context().Value(requestIDKey), r.Method, r.URL.Path, time.Since(start))
	})
}

// GetRequestID retrieves the request ID from a context.
func GetRequestID(ctx context.Context) string {
	if rid, ok := ctx.Value(requestIDKey).(string); ok {
		return rid
	}
	return "unknown"
}

// Placeholder for Sprint 6 Auth middleware
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sprint 6: JWT validation goes here.
		// For Sprint 5, we'll allow everything.
		next.ServeHTTP(w, r)
	})
}
