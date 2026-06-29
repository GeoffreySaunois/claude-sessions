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

// Kind classifies a session by where and how it ran, so noise from automated
// runs can be separated from the user's own interactive work.
type Kind string

const (
	// KindMain is an ordinary interactive session in a real project directory.
	KindMain Kind = "main"
	// KindWorktree is an interactive session inside a Claude Code worktree.
	KindWorktree Kind = "worktree"
	// KindExample ran in a gym examples/ fixture directory.
	KindExample Kind = "example"
	// KindGym ran inside a gym-internal .gym/worktrees directory.
	KindGym Kind = "gym"
	// KindSDK was driven by the Agent SDK (e.g. a gym worker), not the CLI.
	KindSDK Kind = "sdk"
	// KindBackground was launched as a background session.
	KindBackground Kind = "background"
)

// DefaultVisibleKinds are the kinds shown before any filter: the user's own
// interactive work. Automated/fixture runs are hidden until explicitly shown.
func DefaultVisibleKinds() map[Kind]bool {
	return map[Kind]bool{KindMain: true, KindWorktree: true, KindBackground: true}
}

// Session is one Claude Code conversation: its on-disk transcript, the context
// it ran in, its current live status, and the user's organization metadata.
type Session struct {
	ID          string    `json:"id"`          // sessionId == transcript filename stem
	Path        string    `json:"path"`        // absolute path to the .jsonl transcript
	ProjectDir  string    `json:"projectDir"`  // ~/.claude/projects/<slug>
	Cwd         string    `json:"cwd"`         // working directory the session ran in
	GitBranch   string    `json:"gitBranch"`   // git branch at session start, if any
	Title       string    `json:"title"`       // custom title > ai title > first prompt
	LastMessage string    `json:"lastMessage"` // preview of the most recent message text
	Kind        Kind      `json:"kind"`        // classification (main / sdk / example / …)
	Status      Status    `json:"status"`
	PID         int       `json:"pid"`        // owning live process, 0 if none
	LastActive  time.Time `json:"lastActive"` // transcript mtime
	SizeBytes   int64     `json:"sizeBytes"`
	Version     string    `json:"version"` // Claude Code version that wrote the transcript

	// User-maintained organization metadata (sidecar, never from ~/.claude).
	Pinned   bool     `json:"pinned"` // adopted into the curated dashboard
	Category string   `json:"category"`
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
