package tui

import (
	"fmt"

	"claude-sessions/core"

	tea "github.com/charmbracelet/bubbletea"
)

// sessionsLoadedMsg carries the result of an asynchronous refresh.
type sessionsLoadedMsg struct {
	sessions []core.Session
	err      error
}

// Update routes input to the list handler or the active text-input overlay.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.scrollToCursor()
		return m, nil
	case sessionsLoadedMsg:
		return m.applyRefresh(msg), nil
	case tea.KeyMsg:
		if m.mode == modeList {
			return m.updateList(msg)
		}
		return m.updateInput(msg)
	}
	return m, nil
}

// applyRefresh swaps in freshly loaded sessions, dropping picks that vanished.
func (m Model) applyRefresh(msg sessionsLoadedMsg) Model {
	if msg.err != nil {
		m.status = "refresh failed: " + msg.err.Error()
		return m
	}
	m.sessions = msg.sessions
	m.prunePicks()
	m.rebuildRows()
	m.status = fmt.Sprintf("refreshed %d sessions", len(msg.sessions))
	return m
}

// prunePicks drops selections whose session is no longer loaded.
func (m *Model) prunePicks() {
	live := map[string]bool{}
	for _, s := range m.sessions {
		live[s.ID] = true
	}
	for id := range m.picked {
		if !live[id] {
			delete(m.picked, id)
		}
	}
}

// updateList handles keys in the default list mode.
func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		m.moveCursor(1)
	case "k", "up":
		m.moveCursor(-1)
	case " ":
		m.togglePick()
	case "o":
		return m, m.openSelected()
	case "/":
		m.enterFilter()
	case "esc":
		m.clearFilter()
	case "a":
		m.toggleArchiveSelected()
	case "t":
		m.enterTagEdit()
	case "f":
		m.enterCategoryEdit()
	case "s":
		m.cycleSort()
	case "H":
		m.toggleArchivedVisibility()
	case "T":
		m.cycleTheme()
	case "h", "?":
		m.help = !m.help
		m.scrollToCursor()
	case "r":
		return m, m.refresh()
	}
	return m, nil
}

// moveCursor steps the cursor by delta, skipping header rows in the travel
// direction and clamping to the list bounds.
func (m *Model) moveCursor(delta int) {
	if len(m.rows) == 0 {
		return
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	i := m.cursor
	for n := 0; n < abs(delta); n++ {
		i = m.skipHeaders(i, step)
	}
	m.cursor = i
	m.scrollToCursor()
}

// skipHeaders moves one logical step from i in the given direction, advancing
// past any header rows but never off the ends of the list.
func (m Model) skipHeaders(i, step int) int {
	for {
		next := i + step
		if next < 0 || next >= len(m.rows) {
			return i // stay put at the boundary
		}
		i = next
		if !m.rows[i].isHeader() {
			return i
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// togglePick toggles the cursor session's membership in the selection.
func (m *Model) togglePick() {
	s, ok := m.cursorSession()
	if !ok {
		return
	}
	if m.picked[s.ID] {
		delete(m.picked, s.ID)
	} else {
		m.picked[s.ID] = true
	}
}

// openSelected returns a command that opens the selected sessions as Ghostty
// splits, reporting the outcome via the status line.
func (m *Model) openSelected() tea.Cmd {
	sel := m.selectedSessions()
	if len(sel) == 0 {
		m.status = "nothing to open"
		return nil
	}
	m.status = fmt.Sprintf("opening %d session(s)…", len(sel))
	return func() tea.Msg {
		_ = core.Open(sel, core.DefaultOpenConfig())
		return nil
	}
}

// cycleSort advances the sort/grouping mode and rebuilds the rows.
func (m *Model) cycleSort() {
	m.sort = m.sort.next()
	m.rebuildRows()
	m.status = "sort: " + m.sort.label()
}

// toggleArchivedVisibility shows or hides archived sessions.
func (m *Model) toggleArchivedVisibility() {
	m.showArchived = !m.showArchived
	m.rebuildRows()
	if m.showArchived {
		m.status = "showing archived"
	} else {
		m.status = "hiding archived"
	}
}

// toggleArchiveSelected flips the archived flag of every selected session and
// persists each change, then refreshes the derived rows.
func (m *Model) toggleArchiveSelected() {
	sel := m.selectedSessions()
	if len(sel) == 0 {
		return
	}
	for _, s := range sel {
		want := !s.Archived
		if err := m.store.Update(s.ID, func(meta *core.SessionMeta) { meta.Archived = want }); err != nil {
			m.status = "archive failed: " + err.Error()
			return
		}
		m.setSessionArchived(s.ID, want)
	}
	m.rebuildRows()
	m.status = fmt.Sprintf("archived/unarchived %d session(s)", len(sel))
}

// setSessionArchived mirrors a persisted archived change into the in-memory
// session so the view updates without a full reload.
func (m *Model) setSessionArchived(id string, archived bool) {
	for i := range m.sessions {
		if m.sessions[i].ID == id {
			m.sessions[i].Archived = archived
			return
		}
	}
}

// refresh returns a command that reloads all sessions from disk.
func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg {
		sessions, err := core.LoadSessions()
		return sessionsLoadedMsg{sessions: sessions, err: err}
	}
}
