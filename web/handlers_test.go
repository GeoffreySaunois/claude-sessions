package web

import (
	"testing"

	"claude-sessions/core"
)

// selectByIDs resolves the POST /api/open body against live sessions. A bug
// here would open the wrong sessions (or the right ones in the wrong split
// order), so we check order-preservation and silent dropping of unknown ids.
func TestSelectByIDs(t *testing.T) {
	sessions := []core.Session{
		{ID: "a", Title: "A"},
		{ID: "b", Title: "B"},
		{ID: "c", Title: "C"},
	}

	got := selectByIDs(sessions, []string{"c", "missing", "a"})

	if len(got) != 2 {
		t.Fatalf("want 2 matched, got %d", len(got))
	}
	if got[0].ID != "c" || got[1].ID != "a" {
		t.Fatalf("order not preserved: got %q, %q", got[0].ID, got[1].ID)
	}
}

func TestSelectByIDsEmpty(t *testing.T) {
	if got := selectByIDs(nil, []string{"x"}); len(got) != 0 {
		t.Fatalf("want empty, got %d", len(got))
	}
}

// applyMetaPatch must touch ONLY the fields present in the request. The whole
// point of the pointer fields is that an archive toggle doesn't wipe tags and a
// tag edit doesn't reset the category. A regression here silently destroys user
// metadata, so we exercise each "absent field is preserved" case.
func TestApplyMetaPatchOnlyTouchesPresentFields(t *testing.T) {
	str := func(s string) *string { return &s }
	tagsP := func(t ...string) *[]string { return &t }
	boolP := func(b bool) *bool { return &b }

	t.Run("archive toggle preserves category and tags", func(t *testing.T) {
		m := core.SessionMeta{Category: "work", Tags: []string{"x", "y"}, Archived: false}
		applyMetaPatch(metaRequest{ID: "s", Archived: boolP(true)})(&m)
		if !m.Archived {
			t.Fatal("archived not applied")
		}
		if m.Category != "work" {
			t.Fatalf("category clobbered: %q", m.Category)
		}
		if len(m.Tags) != 2 {
			t.Fatalf("tags clobbered: %v", m.Tags)
		}
	})

	t.Run("tags edit preserves category and archived", func(t *testing.T) {
		m := core.SessionMeta{Category: "work", Tags: []string{"old"}, Archived: true}
		applyMetaPatch(metaRequest{ID: "s", Tags: tagsP("new")})(&m)
		if len(m.Tags) != 1 || m.Tags[0] != "new" {
			t.Fatalf("tags not applied: %v", m.Tags)
		}
		if m.Category != "work" || !m.Archived {
			t.Fatalf("other fields clobbered: cat=%q arch=%v", m.Category, m.Archived)
		}
	})

	t.Run("clearing category sends empty string, not absent", func(t *testing.T) {
		m := core.SessionMeta{Category: "work", Tags: []string{"x"}}
		applyMetaPatch(metaRequest{ID: "s", Category: str("")})(&m)
		if m.Category != "" {
			t.Fatalf("category not cleared: %q", m.Category)
		}
		if len(m.Tags) != 1 {
			t.Fatalf("tags clobbered while clearing category: %v", m.Tags)
		}
	})
}

// bulkMutator dispatches the bulk action to the right SessionMeta change. A
// wrong dispatch (e.g. archive setting unarchive, or category not applying the
// value) would corrupt many sessions at once, so we check each branch and that
// an unknown action errors instead of silently no-op-ing.
func TestBulkMutator(t *testing.T) {
	t.Run("archive sets archived true without touching category", func(t *testing.T) {
		fn, err := bulkMutator(bulkRequest{Action: "archive"})
		if err != nil {
			t.Fatal(err)
		}
		m := core.SessionMeta{Category: "keep", Archived: false}
		fn(&m)
		if !m.Archived || m.Category != "keep" {
			t.Fatalf("archive wrong: arch=%v cat=%q", m.Archived, m.Category)
		}
	})

	t.Run("unarchive sets archived false", func(t *testing.T) {
		fn, _ := bulkMutator(bulkRequest{Action: "unarchive"})
		m := core.SessionMeta{Archived: true}
		fn(&m)
		if m.Archived {
			t.Fatal("unarchive did not clear archived")
		}
	})

	t.Run("category applies the request value", func(t *testing.T) {
		fn, _ := bulkMutator(bulkRequest{Action: "category", Value: "moved"})
		m := core.SessionMeta{Category: "old"}
		fn(&m)
		if m.Category != "moved" {
			t.Fatalf("category not applied: %q", m.Category)
		}
	})

	t.Run("unknown action errors", func(t *testing.T) {
		if _, err := bulkMutator(bulkRequest{Action: "delete"}); err == nil {
			t.Fatal("expected error for unknown action")
		}
	})
}
