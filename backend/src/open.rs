//! Open launches selected sessions into the terminal. The default mechanism
//! drives Ghostty's native splits on macOS: there is no CLI to create a split
//! running a command, so we activate Ghostty and inject the new-tab / new-split
//! keybindings followed by the resume command via AppleScript.

use std::fmt::Write as _;
use std::process::Command;

use crate::session::Session;

/// OpenConfig controls how selected sessions are launched into the terminal.
pub struct OpenConfig {
    /// ResumeCommand is the shell run in each surface; `{{cwd}}` and `{{id}}`
    /// are substituted per session.
    pub resume_command: String,
    /// SplitDelay is the AppleScript delay (seconds) after each new surface,
    /// covering the time Ghostty needs to spawn it before it accepts input.
    pub split_delay: f64,
    /// SplitDown lays sessions out in a vertical stack instead of side by side.
    pub split_down: bool,
}

impl Default for OpenConfig {
    /// The Ghostty-native-split configuration.
    fn default() -> Self {
        OpenConfig {
            resume_command: "cd {{cwd}} && claude --resume {{id}}".to_string(),
            split_delay: 0.45,
            split_down: false,
        }
    }
}

/// open launches the given sessions: the first into a fresh Ghostty tab, each
/// subsequent one into a split of that tab, every surface resuming its session.
pub fn open(sessions: &[Session], cfg: &OpenConfig) -> std::io::Result<()> {
    if sessions.is_empty() {
        return Ok(());
    }
    let script = build_ghostty_script(sessions, cfg);
    Command::new("osascript").arg("-e").arg(script).status()?;
    Ok(())
}

/// build_ghostty_script assembles the AppleScript that opens the sessions. It is
/// pure (no side effects) so it can be unit-tested and previewed.
fn build_ghostty_script(sessions: &[Session], cfg: &OpenConfig) -> String {
    let split_key = if cfg.split_down {
        r#"keystroke "d" using {command down, shift down}"#
    } else {
        r#"keystroke "d" using command down"#
    };
    let mut b = String::new();
    b.push_str("tell application \"Ghostty\" to activate\n");
    b.push_str("delay 0.3\n");
    b.push_str("tell application \"System Events\"\n");
    for (i, s) in sessions.iter().enumerate() {
        if i == 0 {
            b.push_str("\tkeystroke \"t\" using command down\n"); // new tab
        } else {
            b.push('\t');
            b.push_str(split_key);
            b.push('\n');
        }
        let _ = writeln!(b, "\tdelay {}", format_delay(cfg.split_delay));
        let _ = writeln!(
            b,
            "\tkeystroke {}",
            applescript_string(&resume_command(cfg, s))
        );
        b.push_str("\tkeystroke return\n");
    }
    b.push_str("end tell\n");
    b
}

fn resume_command(cfg: &OpenConfig, s: &Session) -> String {
    cfg.resume_command
        .replace("{{cwd}}", &shell_quote(&s.cwd))
        .replace("{{id}}", &s.id)
}

/// format_delay renders the delay like Go's `%g`: a compact decimal that drops a
/// trailing `.0`.
fn format_delay(d: f64) -> String {
    if d == d.trunc() {
        format!("{}", d as i64)
    } else {
        let s = format!("{d}");
        s
    }
}

/// applescript_string renders a Rust string as an AppleScript string literal.
fn applescript_string(s: &str) -> String {
    let escaped = s.replace('\\', "\\\\").replace('"', "\\\"");
    format!("\"{escaped}\"")
}

/// shell_quote single-quotes a path for safe use in the resume shell command.
fn shell_quote(s: &str) -> String {
    format!("'{}'", s.replace('\'', r"'\''"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::session::{Kind, Status};

    fn session(id: &str, cwd: &str) -> Session {
        Session {
            id: id.to_string(),
            path: String::new(),
            project_dir: String::new(),
            cwd: cwd.to_string(),
            git_branch: String::new(),
            title: String::new(),
            last_message: String::new(),
            kind: Kind::Main,
            status: Status::Inactive,
            pid: 0,
            last_active: String::new(),
            last_active_nanos: 0,
            size_bytes: 0,
            version: String::new(),
            pinned: false,
            category: String::new(),
            tags: Vec::new(),
            archived: false,
        }
    }

    // The first session opens a new tab; the second uses a split. The resume
    // command must shell-quote the cwd and substitute the id.
    #[test]
    fn script_uses_tab_then_split_and_quotes_cwd() {
        let sessions = vec![session("id1", "/a b"), session("id2", "/c")];
        let script = build_ghostty_script(&sessions, &OpenConfig::default());
        assert!(script.contains("keystroke \"t\" using command down"));
        assert!(script.contains("keystroke \"d\" using command down"));
        assert!(script.contains(r"cd '/a b' && claude --resume id1"));
        assert!(script.contains("delay 0.45"));
    }
}
