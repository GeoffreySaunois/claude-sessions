package tui

import (
	"fmt"
	"strings"
	"time"

	"claude-sessions/core"

	"github.com/charmbracelet/lipgloss"
)

// column widths for the fixed-layout list (project / title / time / branch).
const (
	colProject = 18
	colTitle   = 40
	colTime    = 6
	colBranch  = 20
)

// View renders the whole screen: title bar, status line, the session rows (or
// an empty-state notice), an optional input overlay, and the footer.
func (m Model) View() string {
	s := newStyles()
	var b strings.Builder
	b.WriteString(m.renderTitleBar(s) + "\n")
	if m.status != "" {
		b.WriteString(s.statusMsg.Render(m.status) + "\n")
	}
	b.WriteString(m.renderList(s))
	b.WriteString("\n" + m.renderFooter(s))
	return b.String()
}

// renderTitleBar shows the app name and the active view settings.
func (m Model) renderTitleBar(s styles) string {
	bits := []string{
		fmt.Sprintf("%d sessions", countSessions(m.rows)),
		"sort:" + m.sort.label(),
	}
	if m.filter != "" {
		bits = append(bits, "filter:"+m.filter)
	}
	if m.showArchived {
		bits = append(bits, "+archived")
	}
	left := s.titleBar.Render(" Claude Sessions ")
	right := s.meta.Render(strings.Join(bits, "  ·  "))
	return left + "  " + right
}

// renderList renders every visible row, or an empty-state line, then appends
// the input overlay when one is active.
func (m Model) renderList(s styles) string {
	if len(m.rows) == 0 {
		return s.meta.Render("  (no sessions)") + "\n" + m.renderOverlay(s)
	}
	var b strings.Builder
	for i, r := range m.rows {
		b.WriteString(m.renderRow(s, r, i == m.cursor) + "\n")
	}
	b.WriteString(m.renderOverlay(s))
	return b.String()
}

// renderRow renders a header or a session row, highlighting the cursor row.
func (m Model) renderRow(s styles, r row, cursor bool) string {
	if r.isHeader() {
		return s.header.Render("▸ " + r.header)
	}
	line := m.renderSession(s, r.session)
	if cursor {
		return s.cursorRow.Render(line)
	}
	return line
}

// renderSession lays out one session as status / select-mark / project / title
// / time / branch, dimming archived rows.
func (m Model) renderSession(s styles, sess core.Session) string {
	now := time.Now()
	status := s.statusStyle(sess.Status).Render(statusGlyph(sess.Status) + " " + statusLabel(sess.Status))
	mark := m.renderMark(s, sess)
	project := s.project.Render(pad(projectName(sess), colProject))
	title := s.title.Render(pad(displayTitle(sess), colTitle))
	age := s.meta.Render(pad(relativeTime(sess.LastActive, now), colTime))
	branch := s.meta.Render(pad(sess.GitBranch, colBranch))

	cells := []string{mark, pad(status, 9), project, title, age, branch}
	if tags := renderTags(s, sess); tags != "" {
		cells = append(cells, tags)
	}
	line := strings.Join(cells, " ")
	if sess.Archived {
		return s.archivedRow.Render(line)
	}
	return line
}

// renderMark renders the multi-select checkbox for a session.
func (m Model) renderMark(s styles, sess core.Session) string {
	if m.picked[sess.ID] {
		return s.selectMark.Render("✓")
	}
	return " "
}

// renderTags renders a session's tags as space-separated #labels.
func renderTags(s styles, sess core.Session) string {
	if len(sess.Tags) == 0 {
		return ""
	}
	parts := make([]string, len(sess.Tags))
	for i, t := range sess.Tags {
		parts[i] = "#" + t
	}
	return s.tag.Render(strings.Join(parts, " "))
}

// renderOverlay renders the active text-input prompt, or nothing in list mode.
func (m Model) renderOverlay(s styles) string {
	prompt, ok := overlayPrompt(m.mode)
	if !ok {
		return ""
	}
	return s.filterPrompt.Render(prompt) + m.input.View()
}

// overlayPrompt returns the prompt label for an input mode.
func overlayPrompt(mode inputMode) (string, bool) {
	switch mode {
	case modeFilter:
		return "/", true
	case modeEditTags:
		return "tags (comma-separated): ", true
	case modeEditFolder:
		return "folder: ", true
	default:
		return "", false
	}
}

// renderFooter shows the full help block when expanded, else a compact hint.
func (m Model) renderFooter(s styles) string {
	if m.help {
		return renderHelp(s)
	}
	return s.keyHint.Render("j/k move · space pick · o open · / filter · a archive · t tags · f folder · s sort · H archived · r refresh · ? help · q quit")
}

// renderHelp lists every binding with its description.
func renderHelp(s styles) string {
	pairs := [][2]string{
		{"j / k, ↑/↓", "move cursor"},
		{"space", "toggle select"},
		{"o", "open selected (or cursor) sessions"},
		{"/", "filter by title/project (esc clears)"},
		{"a", "archive / unarchive selected"},
		{"t", "edit tags of cursor session"},
		{"f", "set folder of cursor session"},
		{"s", "cycle sort: last-active / project / folder"},
		{"H", "show / hide archived sessions"},
		{"r", "refresh from disk"},
		{"h / ?", "toggle this help"},
		{"q / ctrl+c", "quit"},
	}
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString(s.helpKey.Render(pad(p[0], 12)) + " " + s.helpDesc.Render(p[1]) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// countSessions counts the session rows (excluding group headers).
func countSessions(rows []row) int {
	n := 0
	for _, r := range rows {
		if !r.isHeader() {
			n++
		}
	}
	return n
}

// displayTitle returns a non-empty title, falling back to a placeholder.
func displayTitle(sess core.Session) string {
	if strings.TrimSpace(sess.Title) == "" {
		return "(untitled)"
	}
	return sess.Title
}

// pad truncates or right-pads s to exactly w display cells.
func pad(s string, w int) string {
	return lipgloss.NewStyle().Width(w).MaxWidth(w).Render(s)
}
