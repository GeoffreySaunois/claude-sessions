// Package core discovers Claude Code sessions on disk, resolves their live
// status, and carries the user-maintained organization metadata (folders,
// tags, archive). It is the shared data layer behind both the TUI and the web
// dashboard — neither frontend reads ~/.claude directly.
package core

import (
	"os"
	"path/filepath"
	"time"
)

// Status is the live state of a session, derived from whether a Claude Code
// process currently owns it and what that process reports.
type Status string

const (
	// StatusBusy means a live process owns the session and Claude is working.
	StatusBusy Status = "busy"
	// StatusWaiting means a live process owns the session but it is idle —
	// the turn is the user's.
	StatusWaiting Status = "waiting"
	// StatusInactive means no live process owns the session.
	StatusInactive Status = "inactive"
)

// Session is one Claude Code conversation: its on-disk transcript, the context
// it ran in, its current live status, and the user's organization metadata.
type Session struct {
	ID         string    `json:"id"`         // sessionId == transcript filename stem
	Path       string    `json:"path"`       // absolute path to the .jsonl transcript
	ProjectDir string    `json:"projectDir"` // ~/.claude/projects/<slug>
	Cwd        string    `json:"cwd"`        // working directory the session ran in
	GitBranch  string    `json:"gitBranch"`  // git branch at session start, if any
	Title      string    `json:"title"`      // custom title > ai title > first prompt
	Status     Status    `json:"status"`
	PID        int       `json:"pid"`        // owning live process, 0 if none
	LastActive time.Time `json:"lastActive"` // transcript mtime
	SizeBytes  int64     `json:"sizeBytes"`
	Version    string    `json:"version"` // Claude Code version that wrote the transcript

	// User-maintained organization metadata (sidecar, never from ~/.claude).
	Folder   string   `json:"folder"`
	Tags     []string `json:"tags"`
	Archived bool     `json:"archived"`
}

// ClaudeDir returns the root Claude Code config directory, honoring
// CLAUDE_CONFIG_DIR and falling back to ~/.claude.
func ClaudeDir() string {
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude"
	}
	return filepath.Join(home, ".claude")
}

func projectsDir() string { return filepath.Join(ClaudeDir(), "projects") }
func sessionsDir() string { return filepath.Join(ClaudeDir(), "sessions") }
