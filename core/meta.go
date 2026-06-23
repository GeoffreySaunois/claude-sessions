package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// SessionMeta is the user-maintained organization data for one session. It
// lives in a sidecar file and never touches Claude Code's own state.
type SessionMeta struct {
	Folder   string   `json:"folder"`
	Tags     []string `json:"tags"`
	Archived bool     `json:"archived"`
}

// MetaStore is the persistent collection of per-session organization metadata,
// keyed by sessionId. It is safe for concurrent use.
type MetaStore struct {
	path string
	mu   sync.Mutex
	data map[string]SessionMeta
}

func metaPath() string {
	return filepath.Join(ClaudeDir(), "session-ui", "meta.json")
}

// LoadMetaStore reads the sidecar file, returning an empty store if none exists.
func LoadMetaStore() (*MetaStore, error) {
	path := metaPath()
	ms := &MetaStore{path: path, data: map[string]SessionMeta{}}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ms, nil
	}
	if err != nil {
		return nil, err
	}
	var doc struct {
		Sessions map[string]SessionMeta `json:"sessions"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if doc.Sessions != nil {
		ms.data = doc.Sessions
	}
	return ms, nil
}

// Get returns the stored metadata for a session, or the zero value if unset.
func (m *MetaStore) Get(id string) SessionMeta {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data[id]
}

// Update applies fn to a session's metadata and persists the store atomically.
func (m *MetaStore) Update(id string, fn func(*SessionMeta)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	meta := m.data[id]
	fn(&meta)
	m.data[id] = meta
	return m.save()
}

// apply overlays stored metadata onto a session.
func (m *MetaStore) apply(s *Session) {
	meta := m.Get(s.ID)
	s.Folder = meta.Folder
	s.Tags = meta.Tags
	s.Archived = meta.Archived
}

// save writes the store via a temp file + rename so a crash never leaves a
// half-written sidecar.
func (m *MetaStore) save() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	doc := struct {
		Sessions map[string]SessionMeta `json:"sessions"`
	}{Sessions: m.data}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}
