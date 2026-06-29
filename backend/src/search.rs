//! Search scans every transcript's conversation text and returns, for each
//! session whose messages contain all query terms (case-insensitive, AND), a
//! short snippet around the first match. Files are scanned concurrently across a
//! bounded thread pool; oversized lines (snapshots, pasted images) are skipped.

use std::collections::HashMap;
use std::fs::File;
use std::io::{BufRead, BufReader};
use std::path::PathBuf;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::thread;

use serde::Deserialize;

use crate::session::projects_dir;

// A snippet keeps a little context before the match and more after it, so the
// matched term sits near the start of the line and stays visible even when the
// UI truncates a long snippet with an ellipsis.
const SNIPPET_LEAD: usize = 24;
const SNIPPET_TRAIL: usize = 180;

const MAX_LINE_BYTES: usize = 256 * 1024;

#[derive(Deserialize, Default)]
struct TranscriptLine {
    #[serde(default)]
    r#type: String,
    #[serde(default)]
    message: Option<TranscriptMsg>,
}

#[derive(Deserialize, Default)]
struct TranscriptMsg {
    #[serde(default)]
    content: serde_json::Value,
}

struct FileRef {
    id: String,
    path: PathBuf,
}

/// search_transcripts scans every transcript and returns id -> snippet for each
/// session matching all whitespace-separated, lowercased query terms (AND).
/// Returns an empty map for an empty query. Intended to run on a blocking pool.
pub fn search_transcripts(query: &str) -> std::io::Result<HashMap<String, String>> {
    let terms: Vec<String> = query
        .to_lowercase()
        .split_whitespace()
        .map(|s| s.to_string())
        .collect();
    if terms.is_empty() {
        return Ok(HashMap::new());
    }
    let files = list_transcript_files()?;
    Ok(scan_files_concurrently(files, &terms))
}

/// scan_files_concurrently fans the per-file scan out across a bounded set of
/// worker threads (one per logical CPU, at least two), each pulling from a
/// shared index, and merges their hits.
fn scan_files_concurrently(files: Vec<FileRef>, terms: &[String]) -> HashMap<String, String> {
    let next = AtomicUsize::new(0);
    let workers = std::thread::available_parallelism()
        .map(|n| n.get())
        .unwrap_or(2)
        .max(2);

    thread::scope(|scope| {
        let mut handles = Vec::with_capacity(workers);
        for _ in 0..workers {
            let next = &next;
            let files = &files;
            handles.push(scope.spawn(move || {
                let mut local: Vec<(String, String)> = Vec::new();
                loop {
                    let i = next.fetch_add(1, Ordering::Relaxed);
                    if i >= files.len() {
                        break;
                    }
                    let f = &files[i];
                    if let Some(snip) = search_file(&f.path, terms) {
                        local.push((f.id.clone(), snip));
                    }
                }
                local
            }));
        }
        let mut out = HashMap::new();
        for h in handles {
            if let Ok(local) = h.join() {
                out.extend(local);
            }
        }
        out
    })
}

/// list_transcript_files enumerates every session transcript across all projects.
fn list_transcript_files() -> std::io::Result<Vec<FileRef>> {
    let root = projects_dir();
    let mut files = Vec::new();
    for entry in std::fs::read_dir(&root)? {
        let entry = match entry {
            Ok(e) => e,
            Err(_) => continue,
        };
        if !entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            continue;
        }
        let dir = entry.path();
        let inner = match std::fs::read_dir(&dir) {
            Ok(d) => d,
            Err(_) => continue,
        };
        for f in inner.flatten() {
            let name = f.file_name();
            let name = name.to_string_lossy();
            if !f.file_type().map(|t| t.is_dir()).unwrap_or(false) && name.ends_with(".jsonl") {
                files.push(FileRef {
                    id: name.strip_suffix(".jsonl").unwrap_or(&name).to_string(),
                    path: dir.join(name.as_ref()),
                });
            }
        }
    }
    Ok(files)
}

/// search_file assembles a transcript's user/assistant message text and reports
/// a snippet if every term is present.
fn search_file(path: &std::path::Path, terms: &[String]) -> Option<String> {
    let f = File::open(path).ok()?;
    let mut r = BufReader::with_capacity(64 * 1024, f);
    let mut text = String::new();
    loop {
        let (raw, too_long, eof) = read_bounded_line(&mut r, MAX_LINE_BYTES);
        if !raw.is_empty() && !too_long {
            if let Ok(line) = serde_json::from_slice::<TranscriptLine>(&raw) {
                if let Some(msg) = &line.message {
                    if line.r#type == "user" || line.r#type == "assistant" {
                        let t = all_message_text(&msg.content);
                        if !t.is_empty() {
                            text.push_str(&t);
                            text.push('\n');
                        }
                    }
                }
            }
        }
        if eof {
            break;
        }
    }

    let lowered = text.to_lowercase();
    for term in terms {
        if !lowered.contains(term.as_str()) {
            return None;
        }
    }
    Some(snippet_around(&text, &lowered, &terms[0]))
}

/// read_bounded_line reads one newline-delimited line, keeping at most `max`
/// bytes; longer lines are drained and flagged. The third value is true at EOF.
fn read_bounded_line<R: BufRead>(r: &mut R, max: usize) -> (Vec<u8>, bool, bool) {
    let mut line: Vec<u8> = Vec::new();
    let mut too_long = false;
    loop {
        let mut chunk = Vec::new();
        let n = match r.read_until(b'\n', &mut chunk) {
            Ok(n) => n,
            Err(_) => {
                trim_eol(&mut line);
                return (line, too_long, true);
            }
        };
        if !too_long && line.len() + chunk.len() <= max {
            line.extend_from_slice(&chunk);
        } else {
            too_long = true;
        }
        if n == 0 || chunk.last() == Some(&b'\n') {
            trim_eol(&mut line);
            return (line, too_long, n == 0);
        }
    }
}

fn trim_eol(line: &mut Vec<u8>) {
    while matches!(line.last(), Some(b'\n') | Some(b'\r')) {
        line.pop();
    }
}

/// all_message_text joins every text block of a message's content, which is
/// either a plain string or an array of typed blocks.
fn all_message_text(raw: &serde_json::Value) -> String {
    if raw.is_null() {
        return String::new();
    }
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
    parts.join(" ")
}

/// snippet_around returns a single-line excerpt centered on the first occurrence
/// of `term`, with ellipses where the text is trimmed. Byte offsets are snapped
/// to UTF-8 char boundaries so the slice never panics.
fn snippet_around(text: &str, lowered: &str, term: &str) -> String {
    let i = match lowered.find(term) {
        Some(i) => i,
        None => return truncate(&collapse_spaces(text), 180),
    };
    let start = floor_char_boundary(text, i.saturating_sub(SNIPPET_LEAD));
    let end = ceil_char_boundary(text, (i + term.len() + SNIPPET_TRAIL).min(text.len()));
    let mut snip = collapse_spaces(&text[start..end]);
    if start > 0 {
        snip = format!("…{snip}");
    }
    if end < text.len() {
        snip.push('…');
    }
    snip
}

fn floor_char_boundary(s: &str, mut i: usize) -> usize {
    if i >= s.len() {
        return s.len();
    }
    while i > 0 && !s.is_char_boundary(i) {
        i -= 1;
    }
    i
}

fn ceil_char_boundary(s: &str, mut i: usize) -> usize {
    if i >= s.len() {
        return s.len();
    }
    while i < s.len() && !s.is_char_boundary(i) {
        i += 1;
    }
    i
}

/// collapse_spaces folds any run of whitespace into a single space so a snippet
/// drawn from a multi-line message renders on one line.
fn collapse_spaces(s: &str) -> String {
    s.split_whitespace().collect::<Vec<_>>().join(" ")
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

    // snippet_around biases the window so the match sits near the start: ~24
    // chars of lead, ~180 of trail, with ellipses marking trimmed ends.
    #[test]
    fn snippet_biases_lead_and_trail() {
        let lead = "x".repeat(60);
        let trail = "y".repeat(300);
        let text = format!("{lead} toffoli {trail}");
        let snip = snippet_around(&text, &text.to_lowercase(), "toffoli");
        assert!(snip.starts_with('…'), "expected leading ellipsis: {snip}");
        assert!(snip.ends_with('…'), "expected trailing ellipsis: {snip}");
        assert!(snip.contains("toffoli"));
        // Lead is bounded near SNIPPET_LEAD chars of 'x' before the term.
        let before = snip.split("toffoli").next().unwrap();
        let xs = before.chars().filter(|c| *c == 'x').count();
        assert!(xs <= SNIPPET_LEAD + 1, "lead too wide: {xs}");
    }
}
