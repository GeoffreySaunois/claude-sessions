package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"claude-sessions/core"
)

// sortMode controls the order and grouping of the session list.
type sortMode int

const (
	// sortByLastActive lists sessions most-recently-active first, ungrouped.
	sortByLastActive sortMode = iota
	// sortByProject groups sessions under their project (base name of Cwd).
	sortByProject
	// sortByFolder groups sessions under their user-assigned folder.
	sortByFolder
)

func (m sortMode) label() string {
	switch m {
	case sortByProject:
		return "project"
	case sortByFolder:
		return "folder"
	default:
		return "last-active"
	}
}

// next cycles last-active -> project -> folder -> last-active.
func (m sortMode) next() sortMode {
	return (m + 1) % 3
}

// projectName is the display name for a session's project: the base name of its
// working directory, or "~" when the cwd is empty.
func projectName(s core.Session) string {
	if s.Cwd == "" {
		return "~"
	}
	return filepath.Base(s.Cwd)
}

// folderName is the display name for a session's folder grouping, falling back
// to a stable bucket when the session has no folder assigned.
func folderName(s core.Session) string {
	if s.Folder == "" {
		return "(no folder)"
	}
	return s.Folder
}

// relativeTime renders the age of t as a compact human string ("now", "3m",
// "2h", "5d", "3w"). Future or zero times render as "now".
func relativeTime(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	}
}

// matchesFilter reports whether a session matches the query as a case-
// insensitive subsequence of either its title or its project name. An empty
// query matches everything.
func matchesFilter(s core.Session, query string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	return subsequence(strings.ToLower(s.Title), q) ||
		subsequence(strings.ToLower(projectName(s)), q)
}

// subsequence reports whether every rune of needle appears in haystack in
// order (a fuzzy match), as opposed to a contiguous substring.
func subsequence(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	n := []rune(needle)
	i := 0
	for _, c := range haystack {
		if c == n[i] {
			i++
			if i == len(n) {
				return true
			}
		}
	}
	return false
}

// filterSessions returns the sessions matching the query, preserving order.
func filterSessions(sessions []core.Session, query string) []core.Session {
	if query == "" {
		return sessions
	}
	out := make([]core.Session, 0, len(sessions))
	for _, s := range sessions {
		if matchesFilter(s, query) {
			out = append(out, s)
		}
	}
	return out
}

// parseTags splits a comma-separated tag string into trimmed, non-empty,
// de-duplicated tags, preserving first-seen order.
func parseTags(raw string) []string {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		t := strings.TrimSpace(part)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// row is a single rendered line in the list: either a group header or a session.
type row struct {
	header  string       // non-empty for a group-header row
	session core.Session // valid when header == ""
}

// isHeader reports whether the row is a group header rather than a session.
func (r row) isHeader() bool { return r.header != "" }

// buildRows applies the sort mode to the (already filtered) sessions and
// flattens the result into the displayable rows, inserting group headers in
// grouped modes. Sessions inside a group stay most-recently-active first.
func buildRows(sessions []core.Session, mode sortMode) []row {
	switch mode {
	case sortByProject:
		return groupRows(sessions, projectName)
	case sortByFolder:
		return groupRows(sessions, folderName)
	default:
		rows := make([]row, 0, len(sessions))
		for _, s := range sessions {
			rows = append(rows, row{session: s})
		}
		return rows
	}
}

// groupRows buckets sessions by key, orders the groups alphabetically, and
// emits a header row before each group's sessions.
func groupRows(sessions []core.Session, key func(core.Session) string) []row {
	groups := map[string][]core.Session{}
	for _, s := range sessions {
		k := key(s)
		groups[k] = append(groups[k], s)
	}
	names := make([]string, 0, len(groups))
	for k := range groups {
		names = append(names, k)
	}
	sort.Strings(names)
	rows := make([]row, 0, len(sessions)+len(names))
	for _, name := range names {
		rows = append(rows, row{header: name})
		for _, s := range groups[name] {
			rows = append(rows, row{session: s})
		}
	}
	return rows
}

// visibleSessions filters out archived sessions unless showArchived is set.
func visibleSessions(sessions []core.Session, showArchived bool) []core.Session {
	if showArchived {
		return sessions
	}
	out := make([]core.Session, 0, len(sessions))
	for _, s := range sessions {
		if !s.Archived {
			out = append(out, s)
		}
	}
	return out
}
