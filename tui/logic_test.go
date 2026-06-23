package tui

import (
	"testing"
	"time"

	"claude-sessions/core"
)

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		age  time.Duration
		want string
	}{
		{"zero", 0, "now"},
		{"30s", 30 * time.Second, "now"},
		{"3m", 3 * time.Minute, "3m"},
		{"59m", 59 * time.Minute, "59m"},
		{"2h", 2 * time.Hour, "2h"},
		{"23h", 23 * time.Hour, "23h"},
		{"5d", 5 * 24 * time.Hour, "5d"},
		{"3w", 21 * 24 * time.Hour, "3w"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := relativeTime(now.Add(-c.age), now)
			if got != c.want {
				t.Fatalf("relativeTime(age=%v) = %q, want %q", c.age, got, c.want)
			}
		})
	}
	if got := relativeTime(time.Time{}, now); got != "-" {
		t.Fatalf("zero time = %q, want %q", got, "-")
	}
}

func TestMatchesFilterFuzzy(t *testing.T) {
	s := core.Session{Title: "Fix the auth bug", Cwd: "/home/me/myproject"}
	// Subsequence over the title: f-a-b appears in "Fix the Auth Bug".
	if !matchesFilter(s, "fab") {
		t.Fatal("expected fuzzy subsequence 'fab' to match title")
	}
	// Subsequence over the project name (base of Cwd).
	if !matchesFilter(s, "mypr") {
		t.Fatal("expected 'mypr' to match project name")
	}
	// Order matters: 'baf' is not a subsequence of the title or project.
	if matchesFilter(s, "zzx") {
		t.Fatal("unrelated query should not match")
	}
	// Empty query always matches.
	if !matchesFilter(s, "") {
		t.Fatal("empty query must match everything")
	}
}

func TestMatchesFilterEmptyCwd(t *testing.T) {
	// Must not panic on empty Cwd/Title; project name falls back to "~".
	s := core.Session{}
	if matchesFilter(s, "~") != true {
		t.Fatal("'~' should match the empty-cwd project fallback")
	}
}

func TestParseTags(t *testing.T) {
	got := parseTags("  work , urgent ,, work, done ")
	want := []string{"work", "urgent", "done"}
	if len(got) != len(want) {
		t.Fatalf("parseTags len = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseTags[%d] = %q, want %q (de-dup + order)", i, got[i], want[i])
		}
	}
	if len(parseTags("  , ,")) != 0 {
		t.Fatal("all-empty input should yield no tags")
	}
}

func TestBuildRowsGroupingOrder(t *testing.T) {
	sessions := []core.Session{
		{ID: "1", Cwd: "/x/zeta", Title: "a"},
		{ID: "2", Cwd: "/x/alpha", Title: "b"},
		{ID: "3", Cwd: "/x/alpha", Title: "c"},
	}
	rows := buildRows(sessions, sortByProject)
	// Expect: header(alpha), 2, 3, header(zeta), 1 — groups alphabetical,
	// sessions keep input order within a group.
	wantHeaders := []string{"alpha", "zeta"}
	var gotHeaders []string
	var orderInAlpha []string
	for _, r := range rows {
		if r.isHeader() {
			gotHeaders = append(gotHeaders, r.header)
		} else if projectName(r.session) == "alpha" {
			orderInAlpha = append(orderInAlpha, r.session.ID)
		}
	}
	if len(gotHeaders) != 2 || gotHeaders[0] != wantHeaders[0] || gotHeaders[1] != wantHeaders[1] {
		t.Fatalf("group headers = %v, want %v", gotHeaders, wantHeaders)
	}
	if len(orderInAlpha) != 2 || orderInAlpha[0] != "2" || orderInAlpha[1] != "3" {
		t.Fatalf("alpha group order = %v, want [2 3]", orderInAlpha)
	}
}

func TestBuildRowsLastActiveNoHeaders(t *testing.T) {
	sessions := []core.Session{{ID: "1"}, {ID: "2"}}
	rows := buildRows(sessions, sortByLastActive)
	for _, r := range rows {
		if r.isHeader() {
			t.Fatal("last-active mode must not emit headers")
		}
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
}

func TestVisibleSessionsArchivedFilter(t *testing.T) {
	sessions := []core.Session{
		{ID: "1", Archived: false},
		{ID: "2", Archived: true},
	}
	if got := visibleSessions(sessions, false); len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("hide-archived = %v, want only session 1", got)
	}
	if got := visibleSessions(sessions, true); len(got) != 2 {
		t.Fatalf("show-archived = %d sessions, want 2", len(got))
	}
}

// TestCursorSkipsHeaders reproduces the bug where a fresh model in a grouped
// view would land the cursor on a header row (a non-selectable line).
func TestCursorSkipsHeaders(t *testing.T) {
	store, _ := core.LoadMetaStore()
	sessions := []core.Session{
		{ID: "1", Cwd: "/x/beta"},
		{ID: "2", Cwd: "/x/alpha"},
	}
	m := NewModel(store, sessions, ThemeSystem, true)
	m.cycleSort() // -> project, which inserts headers at index 0
	if _, ok := m.cursorSession(); !ok {
		t.Fatal("cursor must rest on a session, not a header, after grouping")
	}
}

// TestEmptyModelDoesNotPanic guards the empty-list and empty-field edge cases
// across construction and rendering.
func TestEmptyModelDoesNotPanic(t *testing.T) {
	store, _ := core.LoadMetaStore()
	m := NewModel(store, nil, ThemeSystem, true)
	_ = m.View() // must not panic on an empty list

	// A session with empty Cwd/Title/branch must also render cleanly.
	m2 := NewModel(store, []core.Session{{ID: "x"}}, ThemeSystem, true)
	_ = m2.View()
	if s, ok := m2.cursorSession(); !ok || s.ID != "x" {
		t.Fatal("single empty-field session should be selectable")
	}
}
