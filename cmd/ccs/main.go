// Command ccs is the terminal UI for browsing and managing Claude Code
// sessions: it lists every session, surfaces live status, and persists the
// user's folder/tag/archive metadata.
package main

import (
	"fmt"
	"os"

	"claude-sessions/core"
	"claude-sessions/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run loads the session data and drives the Bubbletea program.
func run() error {
	store, err := core.LoadMetaStore()
	if err != nil {
		return err
	}
	sessions, err := core.LoadSessions()
	if err != nil {
		return err
	}
	model := tui.NewModel(store, sessions)
	_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}
