//! Session discovers Claude Code sessions on disk, resolves their live status,
//! and carries the user-maintained organization metadata (folders, tags,
//! archive). It is the shared data layer behind the web dashboard — the
//! frontend never reads `~/.claude` directly.

use std::env;
use std::path::PathBuf;

use serde::Serialize;

/// Status is the live state of a session, derived from whether a Claude Code
/// process currently owns it and what that process reports.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize)]
pub enum Status {
    /// A live process owns the session and Claude is working.
    #[serde(rename = "busy")]
    Busy,
    /// A live process owns the session but it is idle — the turn is the user's.
    #[serde(rename = "waiting")]
    Waiting,
    /// No live process owns the session.
    #[serde(rename = "inactive")]
    Inactive,
}

/// Kind classifies a session by where and how it ran, so noise from automated
/// runs can be separated from the user's own interactive work.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize)]
pub enum Kind {
    /// An ordinary interactive session in a real project directory.
    #[serde(rename = "main")]
    Main,
    /// An interactive session inside a Claude Code worktree.
    #[serde(rename = "worktree")]
    Worktree,
    /// Ran in a gym `examples/` fixture directory.
    #[serde(rename = "example")]
    Example,
    /// Ran inside a gym-internal `.gym/worktrees` directory.
    #[serde(rename = "gym")]
    Gym,
    /// Driven by the Agent SDK (e.g. a gym worker), not the CLI.
    #[serde(rename = "sdk")]
    Sdk,
    /// Launched as a background session.
    #[serde(rename = "background")]
    Background,
}

/// Session is one Claude Code conversation: its on-disk transcript, the context
/// it ran in, its current live status, and the user's organization metadata.
#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Session {
    /// sessionId == transcript filename stem.
    pub id: String,
    /// Absolute path to the `.jsonl` transcript.
    pub path: String,
    /// `~/.claude/projects/<slug>`.
    pub project_dir: String,
    /// Working directory the session ran in.
    pub cwd: String,
    /// Git branch at session start, if any.
    pub git_branch: String,
    /// Custom title > AI title > first prompt.
    pub title: String,
    /// Preview of the most recent message text.
    pub last_message: String,
    /// Classification (main / sdk / example / …).
    pub kind: Kind,
    pub status: Status,
    /// Owning live process, 0 if none.
    pub pid: i32,
    /// Transcript mtime, RFC3339.
    pub last_active: String,
    /// Transcript mtime as nanoseconds since the Unix epoch, used to sort
    /// most-recently-active first without re-parsing the RFC3339 string.
    #[serde(skip)]
    pub last_active_nanos: i128,
    pub size_bytes: u64,
    /// Claude Code version that wrote the transcript.
    pub version: String,

    // User-maintained organization metadata (sidecar, never from ~/.claude).
    /// Adopted into the curated dashboard.
    pub pinned: bool,
    pub category: String,
    pub tags: Vec<String>,
    pub archived: bool,
}

/// claude_dir returns the root Claude Code config directory, honoring
/// `CLAUDE_CONFIG_DIR` and falling back to `~/.claude`.
pub fn claude_dir() -> PathBuf {
    if let Ok(d) = env::var("CLAUDE_CONFIG_DIR") {
        if !d.is_empty() {
            return PathBuf::from(d);
        }
    }
    match home_dir() {
        Some(home) => home.join(".claude"),
        None => PathBuf::from(".claude"),
    }
}

pub fn projects_dir() -> PathBuf {
    claude_dir().join("projects")
}

pub fn sessions_dir() -> PathBuf {
    claude_dir().join("sessions")
}

/// home_dir resolves the user's home directory from the environment, matching
/// the platforms this dashboard targets (Unix `HOME`).
fn home_dir() -> Option<PathBuf> {
    env::var_os("HOME")
        .filter(|h| !h.is_empty())
        .map(PathBuf::from)
}
