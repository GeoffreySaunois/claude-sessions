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
