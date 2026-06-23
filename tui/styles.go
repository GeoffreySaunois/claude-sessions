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

// adaptive pairs a light-background color with a dark-background one; lipgloss
// resolves the right one from the renderer's detected/forced background.
func adaptive(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

func newStyles() styles {
	return styles{
		busy:     lipgloss.NewStyle().Foreground(adaptive("28", "46")).Bold(true), // green
		waiting:  lipgloss.NewStyle().Foreground(adaptive("130", "220")),          // amber
		inactive: lipgloss.NewStyle().Foreground(adaptive("247", "240")),          // dim

		cursorRow:    lipgloss.NewStyle().Background(adaptive("254", "236")).Bold(true),
		selectMark:   lipgloss.NewStyle().Foreground(adaptive("27", "39")).Bold(true), // blue check
		header:       lipgloss.NewStyle().Foreground(adaptive("127", "213")).Bold(true).Underline(true),
		title:        lipgloss.NewStyle().Foreground(adaptive("236", "252")),
		project:      lipgloss.NewStyle().Foreground(adaptive("26", "75")),
		meta:         lipgloss.NewStyle().Foreground(adaptive("245", "244")),
		tag:          lipgloss.NewStyle().Foreground(adaptive("97", "141")),
		archivedRow:  lipgloss.NewStyle().Foreground(adaptive("248", "240")).Italic(true),
		statusMsg:    lipgloss.NewStyle().Foreground(adaptive("29", "48")).Bold(true),
		keyHint:      lipgloss.NewStyle().Foreground(adaptive("245", "244")),
		helpKey:      lipgloss.NewStyle().Foreground(adaptive("27", "39")).Bold(true),
		helpDesc:     lipgloss.NewStyle().Foreground(adaptive("238", "250")),
		filterPrompt: lipgloss.NewStyle().Foreground(adaptive("130", "220")).Bold(true),
		titleBar:     lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(adaptive("62", "57")).Bold(true),
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
