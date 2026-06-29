//! Status reads the per-process session files Claude Code writes under
//! `~/.claude/sessions/` and keeps only those whose process is actually alive,
//! mapping each sessionId to its live state.

use std::collections::HashMap;
use std::fs;

use serde::Deserialize;

use crate::session::{sessions_dir, Status};

/// LiveSession is the subset of a `~/.claude/sessions/<pid>.json` file this
/// module reads. Claude Code writes one such file per running process.
#[derive(Deserialize)]
struct LiveSession {
    #[serde(default)]
    pid: i32,
    #[serde(default, rename = "sessionId")]
    session_id: String,
    /// "busy" while working, "idle" between turns.
    #[serde(default)]
    status: String,
    #[serde(default, rename = "updatedAt")]
    updated_at: i64,
}

/// LiveInfo is the resolved live state of a session keyed by sessionId.
#[derive(Clone, Copy)]
pub struct LiveInfo {
    pub pid: i32,
    pub status: Status,
}

/// resolve_live_statuses reads the per-process session files and keeps only
/// those whose process is actually alive, mapping sessionId to its live state.
/// When several live files name the same session, the most recently updated
/// wins.
pub fn resolve_live_statuses() -> HashMap<String, LiveInfo> {
    let mut out: HashMap<String, LiveInfo> = HashMap::new();
    let mut best: HashMap<String, i64> = HashMap::new(); // sessionId -> chosen updatedAt
    let dir = sessions_dir();
    let entries = match fs::read_dir(&dir) {
        Ok(e) => e,
        Err(_) => return out,
    };
    for entry in entries.flatten() {
        let name = entry.file_name();
        let name = name.to_string_lossy();
        let is_dir = entry.file_type().map(|t| t.is_dir()).unwrap_or(false);
        if is_dir || !name.ends_with(".json") {
            continue;
        }
        let ls = match read_live_session(&entry.path()) {
            Some(ls) => ls,
            None => continue,
        };
        if ls.session_id.is_empty() || !process_alive(ls.pid) {
            continue;
        }
        if let Some(&prev) = best.get(&ls.session_id) {
            if ls.updated_at <= prev {
                continue;
            }
        }
        best.insert(ls.session_id.clone(), ls.updated_at);
        out.insert(
            ls.session_id.clone(),
            LiveInfo {
                pid: ls.pid,
                status: classify(&ls.status),
            },
        );
    }
    out
}

fn read_live_session(path: &std::path::Path) -> Option<LiveSession> {
    let b = fs::read(path).ok()?;
    serde_json::from_slice(&b).ok()
}

/// classify maps Claude Code's reported process status to our Status. Anything
/// that isn't actively working is treated as waiting on the user.
fn classify(raw: &str) -> Status {
    if raw == "busy" {
        Status::Busy
    } else {
        Status::Waiting
    }
}

/// process_alive reports whether a PID names a running process. Signal 0
/// performs existence/permission checks without delivering a signal.
fn process_alive(pid: i32) -> bool {
    if pid <= 0 {
        return false;
    }
    // kill(pid, 0): a process the caller cannot signal (EPERM) returns an error
    // here, which Go's proc.Signal(0) also surfaces — both treat it as not
    // alive for our purposes.
    match rustix::process::Pid::from_raw(pid) {
        Some(p) => rustix::process::test_kill_process(p).is_ok(),
        None => false,
    }
}
