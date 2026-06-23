// Package web serves the Claude Code sessions dashboard: a single embedded
// HTML page plus a small JSON API over the core package. It binds to localhost
// only and is the read/write frontend for browsing, organizing, and opening
// sessions.
package web

import (
	"embed"
	"net/http"

	"claude-sessions/core"
)

//go:embed static/*
var staticFS embed.FS

// Server holds the dependencies and routing for the dashboard.
type Server struct {
	store *core.MetaStore
	mux   *http.ServeMux
}

// NewServer builds a Server with its routes wired and the metadata store loaded.
func NewServer() (*Server, error) {
	store, err := core.LoadMetaStore()
	if err != nil {
		return nil, err
	}
	s := &Server{store: store, mux: http.NewServeMux()}
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
	s.mux.HandleFunc("POST /api/open", s.handleOpen)
	s.mux.HandleFunc("POST /api/meta", s.handleMeta)
}
