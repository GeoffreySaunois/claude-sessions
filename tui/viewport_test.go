package tui

import (
	"fmt"
	"strings"
	"testing"

	"claude-sessions/core"

	tea "github.com/charmbracelet/bubbletea"
)

// makeSessions fabricates n sessions with distinct ids for viewport tests.
func makeSessions(n int) []core.Session {
	out := make([]core.Session, n)
	for i := range out {
		out[i] = core.Session{ID: fmt.Sprintf("s%03d", i), Cwd: "/x/proj", Title: fmt.Sprintf("session %d", i)}
	}
	return out
}

func send(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func keyJ() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")} }

// TestFrameFitsTerminal reproduces the original bug: with far more sessions
// than rows on screen, View must not emit a frame taller than the terminal
// (which made the list "not scroll" and pushed the cursor off-screen).
func TestFrameFitsTerminal(t *testing.T) {
	store, _ := core.LoadMetaStore()
	m := NewModel(store, makeSessions(300), ThemeSystem, true)
	m = send(m, tea.WindowSizeMsg{Width: 100, Height: 20})

	lines := strings.Count(m.View(), "\n") + 1
	if lines > 20 {
		t.Fatalf("frame is %d lines tall but terminal is 20; it must fit", lines)
	}
}

// TestCursorStaysVisible drives the cursor down past one screenful and asserts
// the scroll window follows it, so j is never a no-op off-screen.
func TestCursorStaysVisible(t *testing.T) {
	store, _ := core.LoadMetaStore()
	m := NewModel(store, makeSessions(300), ThemeSystem, true)
	m = send(m, tea.WindowSizeMsg{Width: 100, Height: 20})

	for i := 0; i < 100; i++ {
		m = send(m, keyJ())
	}
	vh := m.viewportHeight()
	if m.cursor != 100 {
		t.Fatalf("cursor = %d, want 100 after 100 downs", m.cursor)
	}
	if m.cursor < m.top || m.cursor >= m.top+vh {
		t.Fatalf("cursor %d outside visible window [%d, %d)", m.cursor, m.top, m.top+vh)
	}
}
