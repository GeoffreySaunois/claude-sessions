package tui

import (
	"strings"

	"claude-sessions/core"

	tea "github.com/charmbracelet/bubbletea"
)

// updateInput handles keys while a text-input overlay (filter / tag / folder)
// is active.
func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.cancelInput(), nil
	case "enter":
		return m.commitInput(), nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.mode == modeFilter {
		m.filter = m.input.Value()
		m.rebuildRows()
	}
	return m, cmd
}

// enterFilter opens the filter overlay seeded with the current query.
func (m *Model) enterFilter() {
	m.mode = modeFilter
	m.input.SetValue(m.filter)
	m.input.Focus()
	m.input.CursorEnd()
}

// enterTagEdit opens the tag editor for the cursor session, seeded with its
// existing comma-joined tags.
func (m *Model) enterTagEdit() {
	s, ok := m.cursorSession()
	if !ok {
		return
	}
	m.mode = modeEditTags
	m.editingID = s.ID
	m.input.SetValue(strings.Join(s.Tags, ", "))
	m.input.Focus()
	m.input.CursorEnd()
}

// enterFolderEdit opens the folder editor for the cursor session, seeded with
// its existing folder.
func (m *Model) enterFolderEdit() {
	s, ok := m.cursorSession()
	if !ok {
		return
	}
	m.mode = modeEditFolder
	m.editingID = s.ID
	m.input.SetValue(s.Folder)
	m.input.Focus()
	m.input.CursorEnd()
}

// cancelInput closes the overlay, discarding tag/folder edits. The filter is
// cleared so the list returns to its unfiltered state.
func (m Model) cancelInput() Model {
	if m.mode == modeFilter {
		m.filter = ""
		m.rebuildRows()
	}
	m.closeInput()
	return m
}

// commitInput applies the overlay's value: filters stay applied, tag/folder
// edits persist to the store and reflect immediately.
func (m Model) commitInput() Model {
	switch m.mode {
	case modeEditTags:
		m.saveTags(parseTags(m.input.Value()))
	case modeEditFolder:
		m.saveFolder(strings.TrimSpace(m.input.Value()))
	}
	m.closeInput()
	return m
}

// closeInput returns to list mode and blurs the shared text input.
func (m *Model) closeInput() {
	m.mode = modeList
	m.editingID = ""
	m.input.Blur()
}

// clearFilter resets any active filter from list mode (the `esc` shortcut).
func (m *Model) clearFilter() {
	if m.filter == "" {
		return
	}
	m.filter = ""
	m.rebuildRows()
	m.status = "filter cleared"
}

// saveTags persists new tags for the editing session and mirrors them in memory.
func (m *Model) saveTags(tags []string) {
	id := m.editingID
	if err := m.store.Update(id, func(meta *core.SessionMeta) { meta.Tags = tags }); err != nil {
		m.status = "tag update failed: " + err.Error()
		return
	}
	for i := range m.sessions {
		if m.sessions[i].ID == id {
			m.sessions[i].Tags = tags
			break
		}
	}
	m.rebuildRows()
	m.status = "tags updated"
}

// saveFolder persists a new folder for the editing session and mirrors it in
// memory, then rebuilds rows so folder grouping reflects the change.
func (m *Model) saveFolder(folder string) {
	id := m.editingID
	if err := m.store.Update(id, func(meta *core.SessionMeta) { meta.Folder = folder }); err != nil {
		m.status = "folder update failed: " + err.Error()
		return
	}
	for i := range m.sessions {
		if m.sessions[i].ID == id {
			m.sessions[i].Folder = folder
			break
		}
	}
	m.rebuildRows()
	m.status = "folder updated"
}
