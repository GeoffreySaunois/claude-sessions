//! Open launches selected sessions into the terminal. The default mechanism
//! drives Ghostty's native splits on macOS: there is no CLI to create a split
//! running a command, so we activate Ghostty and inject the new-tab / new-split
//! keybindings followed by the resume command via AppleScript.

use std::fmt::Write as _;
use std::process::Command;

use crate::session::Session;

/// OpenConfig controls how selected sessions are launched into the terminal.
pub struct OpenConfig {
    /// Terminal app to drive via AppleScript.
    pub terminal_app: String,
    /// ResumeCommand is the shell run in each surface; `{{cwd}}` and `{{id}}`
    /// are substituted per session.
    pub resume_command: String,
    /// SplitDelay is the AppleScript delay (seconds) after each new surface,
    /// covering the time the terminal needs to spawn it before accepting input.
    pub split_delay: f64,
    /// AppleScript keystroke fragment that opens a split.
    pub split_keystroke: String,
    /// AppleScript keystroke fragment that rebalances splits, or None to skip.
    pub equalize_keystroke: Option<String>,
}

impl Default for OpenConfig {
    /// The Ghostty-native-split configuration.
    fn default() -> Self {
        OpenConfig {
            terminal_app: "Ghostty".to_string(),
            resume_command: "cd {{cwd}} && claude --resume {{id}}".to_string(),
            split_delay: 0.45,
            split_keystroke: keystroke("cmd+d").unwrap(),
            equalize_keystroke: keystroke("cmd+ctrl+="),
        }
    }
}

impl OpenConfig {
    /// from_config builds the open config from the user config file. The resume
    /// command is `cd <cwd> && <claude_alias> --resume <id>`, so a user only
    /// configures the alias (and the terminal app + split keybindings) — not the
    /// whole command.
    pub fn from_config() -> Self {
        let s = crate::config::load_open_settings();
        OpenConfig {
            terminal_app: s.terminal_app,
            resume_command: format!("cd {{{{cwd}}}} && {} --resume {{{{id}}}}", s.claude_alias),
            split_delay: s.split_delay,
            split_keystroke: keystroke(&s.split_key).unwrap_or_else(|| keystroke("cmd+d").unwrap()),
            equalize_keystroke: keystroke(&s.equalize_key),
        }
    }
}

/// keystroke turns a binding like "cmd+ctrl+=" into the AppleScript fragment
/// `keystroke "=" using {command down, control down}`. The last `+`-segment is
/// the key; earlier ones are modifiers. Returns None for an empty binding.
fn keystroke(binding: &str) -> Option<String> {
    let parts: Vec<&str> = binding
        .split('+')
        .map(str::trim)
        .filter(|p| !p.is_empty())
        .collect();
    let (key, mods) = parts.split_last()?;
    let mods: Vec<&str> = mods
        .iter()
        .filter_map(|m| match m.to_lowercase().as_str() {
            "cmd" | "command" | "⌘" => Some("command down"),
            "ctrl" | "control" | "⌃" => Some("control down"),
            "shift" | "⇧" => Some("shift down"),
            "alt" | "opt" | "option" | "⌥" => Some("option down"),
            _ => None,
        })
        .collect();
    let key = applescript_string(key);
    Some(match mods.as_slice() {
        [] => format!("keystroke {key}"),
        [one] => format!("keystroke {key} using {one}"),
        many => format!("keystroke {key} using {{{}}}", many.join(", ")),
    })
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
    let se = "tell application \"System Events\" to ";
    let mut b = String::new();
    let _ = writeln!(
        b,
        "tell application {} to activate",
        applescript_string(&cfg.terminal_app)
    );
    b.push_str("delay 0.3\n");
    // Each resume command is PASTED (⌘V), not typed: System Events `keystroke`
    // drops characters from long strings, which mangled the command on slower
    // shells. Save the clipboard first so we can restore it afterward.
    b.push_str("set savedClip to \"\"\n");
    b.push_str("try\n\tset savedClip to (the clipboard as text)\nend try\n");
    for s in sessions.iter() {
        // Open a split of the currently-focused pane (no new tab).
        let _ = writeln!(b, "{se}{}", cfg.split_keystroke);
        let _ = writeln!(b, "delay {}", format_delay(cfg.split_delay));
        let _ = writeln!(
            b,
            "set the clipboard to {}",
            applescript_string(&resume_command(cfg, s))
        );
        b.push_str("delay 0.08\n");
        let _ = writeln!(b, "{se}keystroke \"v\" using command down"); // paste
        let _ = writeln!(b, "{se}keystroke return");
        if let Some(eq) = &cfg.equalize_keystroke {
            // rebalance the splits so the panes stay evenly sized.
            let _ = writeln!(b, "{se}{eq}");
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
        assert!(script.contains(r#"tell application "Ghostty" to activate"#));
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

    #[test]
    fn keystroke_parses_bindings() {
        assert_eq!(
            keystroke("cmd+d").unwrap(),
            r#"keystroke "d" using command down"#
        );
        assert_eq!(
            keystroke("cmd+shift+d").unwrap(),
            r#"keystroke "d" using {command down, shift down}"#
        );
        assert_eq!(
            keystroke("cmd+ctrl+=").unwrap(),
            r#"keystroke "=" using {command down, control down}"#
        );
        assert!(keystroke("").is_none()); // empty disables (e.g. equalize off)
    }
}
