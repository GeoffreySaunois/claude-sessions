// Package web serves the Claude Code sessions dashboard: a single embedded
// HTML page plus a small JSON API over the core package. It binds to localhost
// only and is the read/write frontend for browsing, organizing, and opening
// sessions.
package web

import (
	"embed"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

// Server holds the routing for the dashboard. It keeps no long-lived metadata
// store: handlers that read options or mutate load a fresh core.MetaStore so
// they never serve stale state.
type Server struct {
	mux *http.ServeMux
}

// NewServer builds a Server with its routes wired.
func NewServer() (*Server, error) {
	s := &Server{mux: http.NewServeMux()}
	s.routes()
	return s, nil
}

// ServeHTTP dispatches to the server's mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /api/sessions", s.handleSessions)
	s.mux.HandleFunc("GET /api/options", s.handleGetOptions)
	s.mux.HandleFunc("POST /api/options", s.handleAddOption)
	s.mux.HandleFunc("POST /api/meta", s.handleMeta)
	s.mux.HandleFunc("POST /api/bulk", s.handleBulk)
	s.mux.HandleFunc("POST /api/open", s.handleOpen)
}
