//! User configuration read from `<claude-dir>/session-ui/config.toml` (the same
//! directory as the metadata sidecar, honoring `CLAUDE_CONFIG_DIR`).
//!
//! Parsed with a tiny flat `key = value` reader rather than pulling a TOML
//! dependency — the config surface is a handful of scalars. Unknown keys, a
//! missing file, or unparsable values fall back to the defaults.

use std::path::PathBuf;

use crate::session::claude_dir;

/// OpenSettings are the "Open in the terminal" knobs a user can override.
pub struct OpenSettings {
    /// The `claude` command, possibly an alias with flags
    /// (e.g. "claude --dangerously-skip-permissions"). Invoked as
    /// `<claude_alias> --resume <id>`.
    pub claude_alias: String,
    /// Terminal app to drive (AppleScript `tell application "<app>"`).
    pub terminal_app: String,
    /// Keybinding that opens a split, e.g. "cmd+d" / "cmd+shift+d".
    pub split_key: String,
    /// Keybinding that rebalances splits, e.g. "cmd+ctrl+=". Empty disables it.
    pub equalize_key: String,
    /// Seconds to wait after each new split before pasting into it.
    pub split_delay: f64,
}

impl Default for OpenSettings {
    fn default() -> Self {
        OpenSettings {
            claude_alias: "claude".to_string(),
            terminal_app: "Ghostty".to_string(),
            split_key: "cmd+d".to_string(),
            equalize_key: "cmd+ctrl+=".to_string(),
            split_delay: 0.45,
        }
    }
}

fn config_path() -> PathBuf {
    claude_dir().join("session-ui").join("config.toml")
}

/// load_open_settings reads the config file, defaulting if it's absent.
pub fn load_open_settings() -> OpenSettings {
    match std::fs::read_to_string(config_path()) {
        Ok(text) => parse_open_settings(&text),
        Err(_) => OpenSettings::default(),
    }
}

/// parse_open_settings reads flat `key = value` lines, ignoring blanks, `#`
/// comments, and `[section]` headers; unknown keys and invalid values are left
/// at their defaults.
fn parse_open_settings(text: &str) -> OpenSettings {
    let mut s = OpenSettings::default();
    for line in text.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') || line.starts_with('[') {
            continue;
        }
        let Some((key, val)) = line.split_once('=') else {
            continue;
        };
        let val = val.trim().trim_matches('"').trim();
        match key.trim() {
            "claude_alias" if !val.is_empty() => s.claude_alias = val.to_string(),
            "terminal_app" if !val.is_empty() => s.terminal_app = val.to_string(),
            "split_key" if !val.is_empty() => s.split_key = val.to_string(),
            "equalize_key" => s.equalize_key = val.to_string(), // empty = disable
            "split_delay" => {
                if let Ok(f) = val.parse() {
                    s.split_delay = f;
                }
            }
            _ => {}
        }
    }
    s
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_scalars_strips_quotes_and_keeps_defaults() {
        let text =
            "# my config\nclaude_alias = \"cc\"\nsplit_key = \"cmd+shift+d\"\nsplit_delay = 0.7\n";
        let s = parse_open_settings(text);
        assert_eq!(s.claude_alias, "cc"); // quotes stripped
        assert_eq!(s.split_key, "cmd+shift+d");
        assert_eq!(s.split_delay, 0.7);
        assert_eq!(s.terminal_app, "Ghostty"); // absent -> default
        assert_eq!(s.equalize_key, "cmd+ctrl+="); // absent -> default
    }

    #[test]
    fn empty_equalize_key_disables_it_but_empty_config_keeps_defaults() {
        assert_eq!(parse_open_settings("").claude_alias, "claude");
        assert_eq!(
            parse_open_settings("equalize_key = \"\"\n").equalize_key,
            ""
        );
    }
}
