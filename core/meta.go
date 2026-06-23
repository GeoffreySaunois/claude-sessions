package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// SessionMeta is the user-maintained organization data for one session. It
// lives in a sidecar file and never touches Claude Code's own state.
type SessionMeta struct {
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Archived bool     `json:"archived"`
}

// MetaStore is the persistent collection of per-session organization metadata
// plus the universe of category and tag options offered in the UI (so an option
// can exist before any session uses it, Notion-style). Safe for concurrent use.
type MetaStore struct {
	path       string
	mu         sync.Mutex
	data       map[string]SessionMeta
	categories []string
	tags       []string
}

func metaPath() string {
	return filepath.Join(ClaudeDir(), "session-ui", "meta.json")
}

type metaDoc struct {
	Sessions   map[string]SessionMeta `json:"sessions"`
	Categories []string               `json:"categories"`
	Tags       []string               `json:"tags"`
}

// LoadMetaStore reads the sidecar file, returning an empty store if none exists.
// The option universe is seeded from any categories/tags already in use so the
// current data always appears as selectable options.
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
	var doc metaDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if doc.Sessions != nil {
		ms.data = doc.Sessions
	}
	ms.categories = doc.Categories
	ms.tags = doc.Tags
	ms.seedOptionsFromUsage()
	return ms, nil
}

// Get returns the stored metadata for a session, or the zero value if unset.
func (m *MetaStore) Get(id string) SessionMeta {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data[id]
}

// Options returns the category and tag universes for the UI selects, sorted.
func (m *MetaStore) Options() (categories, tags []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.categories...), append([]string(nil), m.tags...)
}

// AddCategory registers a category option without assigning it to any session.
func (m *MetaStore) AddCategory(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.categories = insertSorted(m.categories, name)
	return m.save()
}

// AddTag registers a tag option without assigning it to any session.
func (m *MetaStore) AddTag(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tags = insertSorted(m.tags, name)
	return m.save()
}

// Update applies fn to a session's metadata, folds any newly-used category/tags
// into the option universe, and persists the store atomically.
func (m *MetaStore) Update(id string, fn func(*SessionMeta)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applyUpdate(id, fn)
	return m.save()
}

// UpdateMany applies fn to several sessions and persists once. Used for bulk
// actions (archive all, move to category).
func (m *MetaStore) UpdateMany(ids []string, fn func(*SessionMeta)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		m.applyUpdate(id, fn)
	}
	return m.save()
}

// applyUpdate mutates one session's metadata and registers its options. The
// caller must hold the lock.
func (m *MetaStore) applyUpdate(id string, fn func(*SessionMeta)) {
	meta := m.data[id]
	fn(&meta)
	m.data[id] = meta
	if meta.Category != "" {
		m.categories = insertSorted(m.categories, meta.Category)
	}
	for _, t := range meta.Tags {
		m.tags = insertSorted(m.tags, t)
	}
}

// apply overlays stored metadata onto a session.
func (m *MetaStore) apply(s *Session) {
	meta := m.Get(s.ID)
	s.Category = meta.Category
	s.Tags = meta.Tags
	s.Archived = meta.Archived
}

// seedOptionsFromUsage folds every category/tag currently assigned to a session
// into the option universe. The caller need not hold the lock (load-time only).
func (m *MetaStore) seedOptionsFromUsage() {
	for _, meta := range m.data {
		if meta.Category != "" {
			m.categories = insertSorted(m.categories, meta.Category)
		}
		for _, t := range meta.Tags {
			m.tags = insertSorted(m.tags, t)
		}
	}
}

// save writes the store via a temp file + rename so a crash never leaves a
// half-written sidecar. The caller must hold the lock.
func (m *MetaStore) save() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	doc := metaDoc{Sessions: m.data, Categories: m.categories, Tags: m.tags}
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

// insertSorted adds name to a sorted slice if absent, keeping it sorted.
func insertSorted(xs []string, name string) []string {
	if name == "" {
		return xs
	}
	i := sort.SearchStrings(xs, name)
	if i < len(xs) && xs[i] == name {
		return xs
	}
	return append(xs[:i:i], append([]string{name}, xs[i:]...)...)
}
