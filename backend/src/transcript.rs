//! Transcript reads the tail of one session's `.jsonl` and returns its last N
//! user/assistant turns, chronological, for the conversation-preview modal. Like
//! the discovery scans it reads only the file tail so it stays fast on large
//! transcripts, and skips oversized lines (file snapshots, pasted images).

use std::fs::File;
use std::io::{Read, Seek, SeekFrom};

use serde::{Deserialize, Serialize};

use crate::discover::{find_transcript_path, strip_wrappers};

// Read enough of the tail to comfortably cover `limit` turns; tool-call exchanges
// interleave many short assistant/user lines, so this is generous.
const TAIL_SCAN_BYTES: u64 = 512 * 1024;
const MAX_LINE_BYTES: usize = 256 * 1024;

// Cap each rendered message so a single huge paste can't bloat the response.
const MAX_TEXT_CHARS: usize = 8000;

/// Turn is one rendered conversation message: who spoke, the text (wrappers
/// stripped, newlines preserved, truncated), the names of any tools it invoked,
/// and the line's timestamp when present.
#[derive(Serialize, PartialEq, Debug)]
pub struct Turn {
    pub role: String,
    pub text: String,
    pub ts: String,
    pub tools: Vec<String>,
}

#[derive(Serialize)]
pub struct Transcript {
    pub messages: Vec<Turn>,
}

#[derive(Deserialize, Default)]
struct RawLine {
    #[serde(default)]
    r#type: String,
    #[serde(default)]
    timestamp: String,
    #[serde(default)]
    message: Option<RawMsg>,
}

#[derive(Deserialize, Default)]
struct RawMsg {
    #[serde(default)]
    role: String,
    #[serde(default)]
    content: serde_json::Value,
}

/// read_transcript returns the last `limit` user/assistant turns of the session,
/// chronological. Returns None when the id resolves to no transcript (→ 404).
/// Intended to run on a blocking pool — it does synchronous disk I/O.
pub fn read_transcript(id: &str, limit: usize) -> Option<Transcript> {
    let path = find_transcript_path(id)?;
    let lines = read_tail_lines(&path);
    let mut turns = parse_turns(&lines);
    if turns.len() > limit {
        turns.drain(0..turns.len() - limit);
    }
    Some(Transcript { messages: turns })
}

/// read_tail_lines reads the transcript tail into whole lines, dropping the
/// partial first line left by a mid-file seek. Mirrors `read_last_message`.
fn read_tail_lines(path: &std::path::Path) -> Vec<String> {
    let mut f = match File::open(path) {
        Ok(f) => f,
        Err(_) => return Vec::new(),
    };
    let size = f.metadata().map(|m| m.len()).unwrap_or(0);
    let offset = size.saturating_sub(TAIL_SCAN_BYTES);
    if f.seek(SeekFrom::Start(offset)).is_err() {
        return Vec::new();
    }
    let mut bytes = Vec::new();
    if f.read_to_end(&mut bytes).is_err() {
        return Vec::new();
    }
    let data = String::from_utf8_lossy(&bytes);
    let mut lines: Vec<String> = data.split('\n').map(|s| s.to_string()).collect();
    if offset > 0 && !lines.is_empty() {
        lines.remove(0); // drop the partial first line from the mid-file seek
    }
    lines
}

/// parse_turns extracts the chronological user/assistant turns from raw JSONL
/// lines, keeping a turn only when it carries displayable text or invokes tools
/// (so tool-result-only user lines and bookkeeping rows are skipped).
fn parse_turns(lines: &[String]) -> Vec<Turn> {
    let mut turns = Vec::new();
    for line_str in lines {
        if line_str.len() > MAX_LINE_BYTES {
            continue; // skip oversized lines (snapshots, pasted images)
        }
        if let Some(turn) = parse_turn(line_str) {
            turns.push(turn);
        }
    }
    turns
}

fn parse_turn(line_str: &str) -> Option<Turn> {
    let line: RawLine = serde_json::from_str(line_str).ok()?;
    if line.r#type != "user" && line.r#type != "assistant" {
        return None;
    }
    let msg = line.message?;
    let role = if msg.role.is_empty() {
        line.r#type.clone()
    } else {
        msg.role.clone()
    };
    let text = truncate(
        strip_wrappers(&all_text(&msg.content)).trim(),
        MAX_TEXT_CHARS,
    );
    let tools = tool_names(&msg.content);
    if text.is_empty() && tools.is_empty() {
        return None;
    }
    Some(Turn {
        role,
        text,
        ts: line.timestamp,
        tools,
    })
}

/// all_text joins every text block of a message's content (string or typed-block
/// array) with blank lines between blocks, so multi-block messages stay readable.
fn all_text(raw: &serde_json::Value) -> String {
    if let Some(s) = raw.as_str() {
        return s.to_string();
    }
    let blocks = match raw.as_array() {
        Some(b) => b,
        None => return String::new(),
    };
    let mut parts = Vec::new();
    for b in blocks {
        let ty = b.get("type").and_then(|v| v.as_str()).unwrap_or("");
        let text = b.get("text").and_then(|v| v.as_str()).unwrap_or("");
        if ty == "text" && !text.is_empty() {
            parts.push(text);
        }
    }
    parts.join("\n\n")
}

/// tool_names collects the names of `tool_use` blocks in a message's content.
fn tool_names(raw: &serde_json::Value) -> Vec<String> {
    let blocks = match raw.as_array() {
        Some(b) => b,
        None => return Vec::new(),
    };
    blocks
        .iter()
        .filter(|b| b.get("type").and_then(|v| v.as_str()) == Some("tool_use"))
        .filter_map(|b| b.get("name").and_then(|v| v.as_str()))
        .map(|s| s.to_string())
        .collect()
}

/// truncate caps a string at `n` chars, appending an ellipsis when trimmed.
fn truncate(s: &str, n: usize) -> String {
    if s.chars().count() <= n {
        return s.to_string();
    }
    let truncated: String = s.chars().take(n).collect();
    format!("{truncated}…")
}

#[cfg(test)]
mod tests {
    use super::*;

    // parse_turns is the load-bearing logic: it must keep user/assistant turns in
    // order, strip wrappers, attach tool names, and skip bookkeeping /
    // tool-result-only lines that carry no displayable text.
    #[test]
    fn parse_turns_keeps_text_and_tools_skips_noise() {
        let lines: Vec<String> = vec![
            r#"{"type":"mode"}"#.into(),
            r#"{"type":"user","timestamp":"2026-06-25T13:01:09Z","message":{"role":"user","content":"<system-reminder>noise</system-reminder>\nhello there"}}"#.into(),
            r#"{"type":"assistant","timestamp":"2026-06-25T13:01:12Z","message":{"role":"assistant","content":[{"type":"text","text":"hi"},{"type":"tool_use","name":"Read"},{"type":"tool_use","name":"Bash"}]}}"#.into(),
            // tool_result-only user line: no text, no tool_use → dropped.
            r#"{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}"#.into(),
            r#"{"type":"file-history-snapshot"}"#.into(),
        ];
        let turns = parse_turns(&lines);
        assert_eq!(turns.len(), 2);

        assert_eq!(turns[0].role, "user");
        // strip_wrappers removed the reminder but kept the user's text + newline.
        assert_eq!(turns[0].text, "hello there");
        assert_eq!(turns[0].ts, "2026-06-25T13:01:09Z");
        assert!(turns[0].tools.is_empty());

        assert_eq!(turns[1].role, "assistant");
        assert_eq!(turns[1].text, "hi");
        assert_eq!(turns[1].tools, vec!["Read", "Bash"]);
    }

    // A tool-only assistant message (no text) is still a turn — the modal shows
    // its "⚙ Tool" line — so it must not be dropped.
    #[test]
    fn parse_turns_keeps_tool_only_message() {
        let lines = vec![
            r#"{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit"}]}}"#.into(),
        ];
        let turns = parse_turns(&lines);
        assert_eq!(turns.len(), 1);
        assert_eq!(turns[0].text, "");
        assert_eq!(turns[0].tools, vec!["Edit"]);
    }
}
