package tui

import (
	"claude-sessions/core"

	"github.com/charmbracelet/lipgloss"
)

// styles holds every lipgloss style the view uses, built once so the View
// function never constructs styles inline.
type styles struct {
	busy     lipgloss.Style
	waiting  lipgloss.Style
	inactive lipgloss.Style

	cursorRow    lipgloss.Style
	selectMark   lipgloss.Style
	header       lipgloss.Style
	title        lipgloss.Style
	project      lipgloss.Style
	meta         lipgloss.Style // dim columns: time, branch
	tag          lipgloss.Style
	archivedRow  lipgloss.Style
	statusMsg    lipgloss.Style
	keyHint      lipgloss.Style
	helpKey      lipgloss.Style
	helpDesc     lipgloss.Style
	filterPrompt lipgloss.Style
	titleBar     lipgloss.Style
}

func newStyles() styles {
	return styles{
		busy:     lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true), // bright green
		waiting:  lipgloss.NewStyle().Foreground(lipgloss.Color("220")),           // yellow
		inactive: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),           // dim gray

		cursorRow:    lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true),
		selectMark:   lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true), // blue check
		header:       lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true).Underline(true),
		title:        lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		project:      lipgloss.NewStyle().Foreground(lipgloss.Color("75")),
		meta:         lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		tag:          lipgloss.NewStyle().Foreground(lipgloss.Color("141")),
		archivedRow:  lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true),
		statusMsg:    lipgloss.NewStyle().Foreground(lipgloss.Color("48")).Bold(true),
		keyHint:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		helpKey:      lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		helpDesc:     lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		filterPrompt: lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true),
		titleBar:     lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Bold(true),
	}
}

// statusStyle returns the color style for a status.
func (s styles) statusStyle(st core.Status) lipgloss.Style {
	switch st {
	case core.StatusBusy:
		return s.busy
	case core.StatusWaiting:
		return s.waiting
	default:
		return s.inactive
	}
}

// statusGlyph returns the indicator glyph for a status.
func statusGlyph(st core.Status) string {
	switch st {
	case core.StatusBusy:
		return "●"
	case core.StatusWaiting:
		return "○"
	default:
		return "·"
	}
}

// statusLabel returns the short text label for a status.
func statusLabel(st core.Status) string {
	switch st {
	case core.StatusBusy:
		return "busy"
	case core.StatusWaiting:
		return "wait"
	default:
		return "idle"
	}
}
