package core

import (
	"fmt"
	"os/exec"
	"strings"
)

// OpenConfig controls how selected sessions are launched into the terminal.
//
// The default mechanism drives Ghostty's native splits on macOS: there is no
// CLI to create a split running a command, so we activate Ghostty and inject
// the new-tab / new-split keybindings followed by the resume command via
// AppleScript. SplitDelay covers the time Ghostty needs to spawn each surface
// before it can receive typed input.
type OpenConfig struct {
	// ResumeCommand is the shell run in each surface; {{cwd}} and {{id}} are
	// substituted per session.
	ResumeCommand string
	// SplitDelay is the AppleScript delay (seconds) after each new surface.
	SplitDelay float64
	// SplitDown lays sessions out in a vertical stack instead of side by side.
	SplitDown bool
}

// DefaultOpenConfig returns the Ghostty-native-split configuration.
func DefaultOpenConfig() OpenConfig {
	return OpenConfig{
		ResumeCommand: "cd {{cwd}} && claude --resume {{id}}",
		SplitDelay:    0.45,
		SplitDown:     false,
	}
}

// Open launches the given sessions: the first into a fresh Ghostty tab, each
// subsequent one into a split of that tab, every surface resuming its session.
func Open(sessions []Session, cfg OpenConfig) error {
	if len(sessions) == 0 {
		return nil
	}
	script := buildGhosttyScript(sessions, cfg)
	return exec.Command("osascript", "-e", script).Run()
}

// buildGhosttyScript assembles the AppleScript that opens the sessions. It is
// pure (no side effects) so it can be unit-tested and previewed.
func buildGhosttyScript(sessions []Session, cfg OpenConfig) string {
	splitKey := `keystroke "d" using command down`
	if cfg.SplitDown {
		splitKey = `keystroke "d" using {command down, shift down}`
	}
	var b strings.Builder
	b.WriteString("tell application \"Ghostty\" to activate\n")
	b.WriteString("delay 0.3\n")
	b.WriteString("tell application \"System Events\"\n")
	for i, s := range sessions {
		if i == 0 {
			b.WriteString("\tkeystroke \"t\" using command down\n") // new tab
		} else {
			b.WriteString("\t" + splitKey + "\n")
		}
		fmt.Fprintf(&b, "\tdelay %g\n", cfg.SplitDelay)
		fmt.Fprintf(&b, "\tkeystroke %s\n", appleScriptString(resumeCommand(cfg, s)))
		b.WriteString("\tkeystroke return\n")
	}
	b.WriteString("end tell\n")
	return b.String()
}

func resumeCommand(cfg OpenConfig, s Session) string {
	cmd := strings.ReplaceAll(cfg.ResumeCommand, "{{cwd}}", shellQuote(s.Cwd))
	return strings.ReplaceAll(cmd, "{{id}}", s.ID)
}

// appleScriptString renders a Go string as an AppleScript string literal.
func appleScriptString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return "\"" + s + "\""
}

// shellQuote single-quotes a path for safe use in the resume shell command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
