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
    /// Equalize re-balances all splits (Ghostty ⌘⌃=) after each new one, so the
    /// panes stay evenly sized instead of each split halving the previous pane.
    pub equalize: bool,
}

impl Default for OpenConfig {
    /// The Ghostty-native-split configuration.
    fn default() -> Self {
        OpenConfig {
            resume_command: "cd {{cwd}} && claude --resume {{id}}".to_string(),
            split_delay: 0.45,
            split_down: false,
            equalize: true,
        }
    }
}

impl OpenConfig {
    /// from_env is the default, with the resume command overridden by
    /// `CCS_RESUME_COMMAND` when set — so users can point it at their own alias,
    /// e.g. `cd {{cwd}} && cc --resume {{id}}` (a `cc` that adds bypass flags).
    /// The override must keep the `{{cwd}}` and `{{id}}` placeholders.
    pub fn from_env() -> Self {
        let mut cfg = OpenConfig::default();
        if let Ok(cmd) = std::env::var("CCS_RESUME_COMMAND") {
            if !cmd.trim().is_empty() {
                cfg.resume_command = cmd;
            }
        }
        cfg
    }
}

/// open launches the given sessions: each one into a split of the currently
/// focused Ghostty pane, every surface resuming its session.
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
    // command-down, plus shift for a vertical stack.
    let split_mods = if cfg.split_down {
        "{command down, shift down}"
    } else {
        "command down"
    };
    let se = "tell application \"System Events\" to ";
    let mut b = String::new();
    b.push_str("tell application \"Ghostty\" to activate\n");
    b.push_str("delay 0.3\n");
    // Each resume command is PASTED (⌘V), not typed: System Events `keystroke`
    // drops characters from long strings, which mangled the command on slower
    // shells. Save the clipboard first so we can restore it afterward.
    b.push_str("set savedClip to \"\"\n");
    b.push_str("try\n\tset savedClip to (the clipboard as text)\nend try\n");
    for s in sessions.iter() {
        // Open a split of the currently-focused pane (no new tab).
        let _ = writeln!(b, "{se}keystroke \"d\" using {split_mods}");
        let _ = writeln!(b, "delay {}", format_delay(cfg.split_delay));
        let _ = writeln!(
            b,
            "set the clipboard to {}",
            applescript_string(&resume_command(cfg, s))
        );
        b.push_str("delay 0.08\n");
        let _ = writeln!(b, "{se}keystroke \"v\" using command down"); // paste
        let _ = writeln!(b, "{se}keystroke return");
        if cfg.equalize {
            // ⌘⌃= — rebalance the splits so the panes stay evenly sized.
            let _ = writeln!(
                b,
                "{se}keystroke \"=\" using {{command down, control down}}"
            );
        }
    }
    b.push_str("delay 0.1\n");
    b.push_str("try\n\tset the clipboard to savedClip\nend try\n");
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

    // Every session opens as a split (⌘D), never a new tab. The resume command
    // must shell-quote the cwd and substitute the id.
    #[test]
    fn script_uses_splits_only_and_quotes_cwd() {
        let sessions = vec![session("id1", "/a b"), session("id2", "/c")];
        let script = build_ghostty_script(&sessions, &OpenConfig::default());
        assert!(!script.contains("keystroke \"t\""));
        assert_eq!(
            script.matches("keystroke \"d\" using command down").count(),
            2
        );
        // The command is pasted (clipboard + ⌘V), never typed char-by-char.
        assert!(script.contains(r#"set the clipboard to "cd '/a b' && claude --resume id1""#));
        assert!(!script.contains("keystroke \"cd"));
        assert_eq!(
            script.matches("keystroke \"v\" using command down").count(),
            2
        );
        assert!(script.contains("delay 0.45"));
        // Splits are equalized after each one (⌘⌃=).
        assert_eq!(
            script
                .matches(r#"keystroke "=" using {command down, control down}"#)
                .count(),
            2
        );
    }
}
