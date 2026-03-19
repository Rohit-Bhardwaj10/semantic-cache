package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Rohit-Bhardwaj10/semantic-cache/internal/cache"
)

// Server defines the HTTP server that hosts the semantic cache API.
type Server struct {
	httpServer *http.Server
	mux        *http.ServeMux
	handler    *Handler
}

func NewServer(addr string, coord *cache.Coordinator) *Server {
	h := NewHandler(coord)
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/cache/query", h.HandleQuery)
	mux.HandleFunc("/health", h.HandleHealth)
	mux.HandleFunc("/readyz", h.HandleHealth)
	mux.HandleFunc("/livez", h.HandleHealth)
	mux.HandleFunc("/analytics/cost-savings", h.HandleAnalytics)

	// Wrap in middleware chain (applied in reverse order)
	var finalHandler http.Handler = mux
	finalHandler = AuthMiddleware(finalHandler)
	finalHandler = LoggerMiddleware(finalHandler)
	finalHandler = RequestIDMiddleware(finalHandler)

	srv := &http.Server{
		Addr:         addr,
		Handler:      finalHandler,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &Server{
		httpServer: srv,
		mux:        mux,
		handler:    h,
	}
}

// Start runs the server until manually stopped or a fatal error occurs.
func (s *Server) Start() error {
	log.Printf("Starting HTTP server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown initiates a graceful close of the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}
