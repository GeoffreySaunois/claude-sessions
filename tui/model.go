package tui

import (
	"claude-sessions/core"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// inputMode is the model's interaction mode: the default list, or a text-entry
// overlay (filter / tag edit / folder edit).
type inputMode int

const (
	modeList inputMode = iota
	modeFilter
	modeEditTags
	modeEditFolder
)

// Model is the full TUI state: the loaded sessions, the user's view settings
// (sort, archived visibility, filter), the multi-selection, the cursor, and
// the current input mode plus its text input.
type Model struct {
	store    *core.MetaStore
	sessions []core.Session // all sessions as last loaded, newest-active first

	rows   []row           // derived: visible + filtered + grouped, rebuilt on change
	cursor int             // index into rows; always points at a session row when possible
	top    int             // index of the first row shown in the scroll window
	picked map[string]bool // selected session IDs

	mode         inputMode
	filter       string
	sort         sortMode
	showArchived bool

	input     textinput.Model // shared by filter / tag / folder modes
	editingID string          // session being edited in tag/folder mode

	status string // transient one-line message shown under the title
	help   bool   // whether the full help footer is expanded

	theme      Theme
	systemDark bool // terminal background darkness detected at startup

	width, height int
}

// NewModel builds the initial model from a metadata store and the sessions to
// display. The theme and detected background darkness are supplied by the
// caller so construction performs no terminal I/O and stays safe in tests.
func NewModel(store *core.MetaStore, sessions []core.Session, theme Theme, systemDark bool) Model {
	in := textinput.New()
	in.Prompt = ""
	m := Model{
		store:      store,
		sessions:   sessions,
		picked:     map[string]bool{},
		mode:       modeList,
		sort:       sortByLastActive,
		input:      in,
		theme:      theme,
		systemDark: systemDark,
		width:      80,
		height:     24,
	}
	applyTheme(theme, systemDark)
	m.rebuildRows()
	return m
}

// Init satisfies tea.Model; the TUI starts with no pending command.
func (m Model) Init() tea.Cmd { return nil }

// rebuildRows recomputes the visible rows from the current sessions, archived
// visibility, filter, and sort mode, then re-anchors the cursor onto a session.
func (m *Model) rebuildRows() {
	visible := visibleSessions(m.sessions, m.showArchived)
	filtered := filterSessions(visible, m.filter)
	m.rows = buildRows(filtered, m.sort)
	m.clampCursor()
}

// clampCursor keeps the cursor in range and off group-header rows when a
// session row is reachable.
func (m *Model) clampCursor() {
	if len(m.rows) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.rows[m.cursor].isHeader() {
		m.moveCursor(1)
	}
	m.scrollToCursor()
}

// viewportHeight is how many list rows fit on screen, given the title bar,
// optional status line, optional input overlay, and footer.
func (m Model) viewportHeight() int {
	chrome := 1 // title bar
	if m.status != "" {
		chrome++
	}
	if _, ok := overlayPrompt(m.mode); ok {
		chrome++
	}
	chrome++ // blank line before the footer
	chrome += m.footerLines()
	v := m.height - chrome
	if v < 1 {
		return 1
	}
	return v
}

// scrollToCursor adjusts the scroll window so the cursor row stays visible.
func (m *Model) scrollToCursor() {
	vh := m.viewportHeight()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+vh {
		m.top = m.cursor - vh + 1
	}
	if maxTop := len(m.rows) - vh; m.top > maxTop {
		m.top = maxTop
	}
	if m.top < 0 {
		m.top = 0
	}
}

// cycleTheme advances the theme and applies it immediately.
func (m *Model) cycleTheme() {
	m.theme = m.theme.next()
	applyTheme(m.theme, m.systemDark)
	m.status = "theme: " + m.theme.label()
}

// cursorSession returns the session under the cursor and whether one exists
// (false on an empty list or a header row).
func (m Model) cursorSession() (core.Session, bool) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return core.Session{}, false
	}
	r := m.rows[m.cursor]
	if r.isHeader() {
		return core.Session{}, false
	}
	return r.session, true
}

// selectedSessions returns every picked session in display order. When nothing
// is picked it returns the cursor session alone (or empty).
func (m Model) selectedSessions() []core.Session {
	var out []core.Session
	for _, r := range m.rows {
		if r.isHeader() {
			continue
		}
		if m.picked[r.session.ID] {
			out = append(out, r.session)
		}
	}
	if len(out) > 0 {
		return out
	}
	if s, ok := m.cursorSession(); ok {
		return []core.Session{s}
	}
	return nil
}
