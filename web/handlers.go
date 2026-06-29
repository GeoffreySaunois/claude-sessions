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

// handleSearch runs a full-text search over every transcript's conversation and
// returns a map of session id to a snippet around the first match.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	matches, err := core.SearchTranscripts(r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]map[string]string{"matches": matches})
}

// optionsResponse is the body of GET /api/options: the universe of category
// and tag options the user has created or used.
type optionsResponse struct {
	Categories []string `json:"categories"`
	Tags       []string `json:"tags"`
}

func (s *Server) handleGetOptions(w http.ResponseWriter, r *http.Request) {
	store, err := core.LoadMetaStore()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	categories, tags := store.Options()
	writeJSON(w, optionsResponse{Categories: categories, Tags: tags})
}

// addOptionRequest is the body of POST /api/options: register an unassigned
// category or tag into the option universe.
type addOptionRequest struct {
	Kind string `json:"kind"` // "category" | "tag"
	Name string `json:"name"`
}

func (s *Server) handleAddOption(w http.ResponseWriter, r *http.Request) {
	var req addOptionRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	store, err := core.LoadMetaStore()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := addOption(store, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func addOption(store *core.MetaStore, req addOptionRequest) error {
	switch req.Kind {
	case "category":
		return store.AddCategory(req.Name)
	case "tag":
		return store.AddTag(req.Name)
	default:
		return errBadKind
	}
}

var errBadKind = &apiError{"kind must be category or tag"}

type apiError struct{ msg string }

func (e *apiError) Error() string { return e.msg }

// metaRequest is the body of POST /api/meta. Each field is a pointer so the
// handler can tell "absent" from "set to zero value" and only apply the fields
// the client actually sent — an archive toggle must not wipe tags.
type metaRequest struct {
	ID       string    `json:"id"`
	Category *string   `json:"category"`
	Tags     *[]string `json:"tags"`
	Archived *bool     `json:"archived"`
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
	store, err := core.LoadMetaStore()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := store.Update(req.ID, applyMetaPatch(req)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// applyMetaPatch returns a mutator that sets only the fields present in req,
// leaving every absent field untouched.
func applyMetaPatch(req metaRequest) func(*core.SessionMeta) {
	return func(m *core.SessionMeta) {
		if req.Category != nil {
			m.Category = *req.Category
		}
		if req.Tags != nil {
			m.Tags = *req.Tags
		}
		if req.Archived != nil {
			m.Archived = *req.Archived
		}
	}
}

// pinRequest is the body of POST /api/pin: adopt a session into the curated
// dashboard, or remove it (which also clears its category/tags/archived).
type pinRequest struct {
	ID     string `json:"id"`
	Pinned bool   `json:"pinned"`
}

func (s *Server) handlePin(w http.ResponseWriter, r *http.Request) {
	var req pinRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	store, err := core.LoadMetaStore()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := store.SetPinned(req.ID, req.Pinned); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// bulkRequest is the body of POST /api/bulk: one action applied to many ids.
type bulkRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"` // "pin" | "unpin" | "archive" | "unarchive" | "category"
	Value  string   `json:"value"`  // category name when action == "category"
}

func (s *Server) handleBulk(w http.ResponseWriter, r *http.Request) {
	var req bulkRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	store, err := core.LoadMetaStore()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := applyBulk(store, req); err != nil {
		http.Error(w, err.Error(), bulkErrorStatus(err))
		return
	}
	writeJSON(w, map[string]int{"updated": len(req.IDs)})
}

// applyBulk dispatches a bulk action. Pin/unpin adopt or release sessions (and
// unpin clears their organization), so they go through SetPinnedMany rather
// than a metadata mutator; the remaining actions edit existing metadata.
func applyBulk(store *core.MetaStore, req bulkRequest) error {
	switch req.Action {
	case "pin":
		return store.SetPinnedMany(req.IDs, true)
	case "unpin":
		return store.SetPinnedMany(req.IDs, false)
	default:
		mutate, err := bulkMutator(req)
		if err != nil {
			return err
		}
		return store.UpdateMany(req.IDs, mutate)
	}
}

// bulkErrorStatus reports 400 for an unknown action and 500 for a store
// failure, so a bad request and a server fault are distinguishable.
func bulkErrorStatus(err error) int {
	if _, ok := err.(*apiError); ok {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// bulkMutator dispatches a metadata-editing bulk action to the SessionMeta
// mutator that applies it, rejecting unknown actions.
func bulkMutator(req bulkRequest) (func(*core.SessionMeta), error) {
	switch req.Action {
	case "archive":
		return func(m *core.SessionMeta) { m.Archived = true }, nil
	case "unarchive":
		return func(m *core.SessionMeta) { m.Archived = false }, nil
	case "category":
		value := req.Value
		return func(m *core.SessionMeta) { m.Category = value }, nil
	default:
		return nil, &apiError{"unknown action: " + req.Action}
	}
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
