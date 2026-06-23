package tui

import "github.com/charmbracelet/lipgloss"

// Theme selects the color palette's light/dark resolution. System follows the
// terminal's detected background; Light and Dark force the choice.
type Theme int

const (
	ThemeSystem Theme = iota
	ThemeLight
	ThemeDark
)

// ParseTheme maps a flag value to a Theme, defaulting to System.
func ParseTheme(s string) Theme {
	switch s {
	case "light":
		return ThemeLight
	case "dark":
		return ThemeDark
	default:
		return ThemeSystem
	}
}

func (t Theme) label() string {
	switch t {
	case ThemeLight:
		return "light"
	case ThemeDark:
		return "dark"
	default:
		return "system"
	}
}

// next cycles system -> light -> dark -> system.
func (t Theme) next() Theme { return (t + 1) % 3 }

// applyTheme points lipgloss's adaptive colors at the right palette by setting
// the default renderer's background darkness. System uses the baseline detected
// from the terminal at startup.
func applyTheme(t Theme, systemDark bool) {
	switch t {
	case ThemeLight:
		lipgloss.SetHasDarkBackground(false)
	case ThemeDark:
		lipgloss.SetHasDarkBackground(true)
	default:
		lipgloss.SetHasDarkBackground(systemDark)
	}
}
