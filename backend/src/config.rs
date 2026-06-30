//! User configuration read from `<claude-dir>/session-ui/config.toml` (the same
//! directory as the metadata sidecar, honoring `CLAUDE_CONFIG_DIR`).
//!
//! Parsed with a tiny flat `key = value` reader rather than pulling a TOML
//! dependency — the config surface is a handful of scalars. Unknown keys, a
//! missing file, or unparsable values fall back to the defaults.

use std::path::PathBuf;

use crate::session::claude_dir;

/// OpenSettings are the "Open in Ghostty" knobs a user can override.
pub struct OpenSettings {
    /// Program that resumes a session: `<resume_program> --resume <id>`.
    /// Default "claude"; set to your own launcher/alias (e.g. "cc").
    pub resume_program: String,
    /// Rebalance the splits (⌘⌃=) after each new one.
    pub equalize: bool,
    /// Stack splits vertically (⌘⇧D) instead of side by side (⌘D).
    pub split_down: bool,
    /// Seconds to wait after each new split before pasting into it.
    pub split_delay: f64,
}

impl Default for OpenSettings {
    fn default() -> Self {
        OpenSettings {
            resume_program: "claude".to_string(),
            equalize: true,
            split_down: false,
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
            "resume_program" if !val.is_empty() => s.resume_program = val.to_string(),
            "equalize" => {
                if let Ok(b) = val.parse() {
                    s.equalize = b;
                }
            }
            "split_down" => {
                if let Ok(b) = val.parse() {
                    s.split_down = b;
                }
            }
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
            "# my config\n[open]\nresume_program = \"cc\"\nequalize = false\nsplit_delay = 0.7\n";
        let s = parse_open_settings(text);
        assert_eq!(s.resume_program, "cc"); // quotes stripped
        assert!(!s.equalize); // overridden
        assert_eq!(s.split_delay, 0.7);
        assert!(!s.split_down); // absent -> default (false)
    }

    #[test]
    fn empty_config_is_all_defaults() {
        let s = parse_open_settings("");
        assert_eq!(s.resume_program, "claude");
        assert!(s.equalize);
        assert_eq!(s.split_delay, 0.45);
    }
}
