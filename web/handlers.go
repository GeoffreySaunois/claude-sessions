package web

import (
	"encoding/json"
	"net/http"

	"claude-sessions/core"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	page, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := core.LoadSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, sessions)
}

// openRequest is the body of POST /api/open: the session IDs to launch.
type openRequest struct {
	IDs []string `json:"ids"`
}

func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
	var req openRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sessions, err := core.LoadSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	matched := selectByIDs(sessions, req.IDs)
	if err := core.Open(matched, core.DefaultOpenConfig()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]int{"opened": len(matched)})
}

// metaRequest is the body of POST /api/meta: the new organization metadata for
// one session.
type metaRequest struct {
	ID       string   `json:"id"`
	Folder   string   `json:"folder"`
	Tags     []string `json:"tags"`
	Archived bool     `json:"archived"`
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	var req metaRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	err := s.store.Update(req.ID, func(m *core.SessionMeta) {
		m.Folder = req.Folder
		m.Tags = req.Tags
		m.Archived = req.Archived
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// selectByIDs returns the sessions whose ID is in ids, preserving the order of
// ids and skipping any that don't resolve.
func selectByIDs(sessions []core.Session, ids []string) []core.Session {
	byID := make(map[string]core.Session, len(sessions))
	for _, s := range sessions {
		byID[s.ID] = s
	}
	matched := make([]core.Session, 0, len(ids))
	for _, id := range ids {
		if s, ok := byID[id]; ok {
			matched = append(matched, s)
		}
	}
	return matched
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
