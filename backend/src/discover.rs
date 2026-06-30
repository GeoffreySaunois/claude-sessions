//! Discovery scans `~/.claude/projects/*/*.jsonl`, parses each transcript's head
//! for static metadata (cwd, branch, version, titles, kind), and reads a preview
//! of the last message from the tail.

use std::fs::{self, File};
use std::io::{BufRead, BufReader, Read, Seek, SeekFrom};
use std::path::Path;

use serde::Deserialize;

use crate::session::{projects_dir, Kind, Session, Status};

// Metadata (cwd, branch, titles) lives on small message lines, but a transcript
// can open with very large lines — file-history snapshots, pasted-image
// attachments — that a byte budget would be exhausted by before the first
// message. So the head scan is bounded by line count and skips oversized lines
// rather than capping total bytes.
const MAX_META_LINES: usize = 5000;
const MAX_LINE_BYTES: usize = 256 * 1024;

// lastMessageScanBytes caps how much of the transcript tail is read for the
// last-message preview.
const LAST_MESSAGE_SCAN_BYTES: u64 = 128 * 1024;

/// transcript_line is the subset of a transcript JSONL row this module reads.
#[derive(Deserialize, Default)]
struct TranscriptLine {
    #[serde(default)]
    r#type: String,
    #[serde(default)]
    cwd: String,
    #[serde(default, rename = "gitBranch")]
    git_branch: String,
    #[serde(default)]
    version: String,
    /// "cli", "sdk-cli", …
    #[serde(default)]
    entrypoint: String,
    /// "bg", …
    #[serde(default, rename = "sessionKind")]
    session_kind: String,
    /// on ai-title rows
    #[serde(default, rename = "aiTitle")]
    ai_title: String,
    /// on custom-title rows
    #[serde(default, rename = "customTitle")]
    custom_title: String,
    #[serde(default)]
    message: Option<TranscriptMsg>,
}

#[derive(Deserialize, Default)]
struct TranscriptMsg {
    #[serde(default)]
    role: String,
    #[serde(default)]
    content: serde_json::Value,
}

/// discover_transcripts returns every session transcript across all projects,
/// parsed for static metadata. Status and sidecar fields are filled in later.
pub fn discover_transcripts() -> std::io::Result<Vec<Session>> {
    let root = projects_dir();
    let mut sessions = Vec::new();
    for entry in fs::read_dir(&root)? {
        let entry = match entry {
            Ok(e) => e,
            Err(_) => continue,
        };
        if !entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            continue;
        }
        let proj_dir = entry.path();
        let files = match fs::read_dir(&proj_dir) {
            Ok(f) => f,
            Err(_) => continue,
        };
        for f in files.flatten() {
            let name = f.file_name();
            let name = name.to_string_lossy();
            if f.file_type().map(|t| t.is_dir()).unwrap_or(false) || !name.ends_with(".jsonl") {
                continue;
            }
            if let Some(s) = parse_transcript(&proj_dir, &name) {
                sessions.push(s);
            }
        }
    }
    Ok(sessions)
}

/// parse_transcript reads one transcript's head and extracts session metadata.
/// Returns None for an empty stub (no user/assistant message).
fn parse_transcript(proj_dir: &Path, name: &str) -> Option<Session> {
    let path = proj_dir.join(name);
    let info = fs::metadata(&path).ok()?;
    let size = info.len();

    let head = scan_transcript_head(&path);
    if !head.has_messages {
        return None; // empty stub (e.g. a lone bridge-session marker)
    }

    let cwd = if head.cwd.is_empty() {
        cwd_from_slug(proj_dir)
    } else {
        head.cwd
    };
    let kind = classify_kind(&cwd, &head.entrypoint, &head.session_kind);
    let last_message = read_last_message(&path, size);
    let (last_active, last_active_nanos) = mtime(&info);
    let title = if head.title.is_empty() {
        name.strip_suffix(".jsonl").unwrap_or(name).to_string()
    } else {
        head.title
    };

    Some(Session {
        id: name.strip_suffix(".jsonl").unwrap_or(name).to_string(),
        path: path.to_string_lossy().into_owned(),
        project_dir: proj_dir.to_string_lossy().into_owned(),
        cwd,
        git_branch: head.git_branch,
        title,
        last_message,
        kind,
        status: Status::Inactive,
        pid: 0,
        last_active,
        last_active_nanos,
        size_bytes: size,
        version: head.version,
        pinned: false,
        category: String::new(),
        tags: Vec::new(),
        archived: false,
    })
}

/// cwd_from_slug recovers a best-effort working directory from a project
/// directory name, which Claude Code derives from the cwd by replacing path
/// separators with dashes. Used only when a transcript carries no cwd of its own.
fn cwd_from_slug(proj_dir: &Path) -> String {
    proj_dir
        .file_name()
        .map(|n| n.to_string_lossy().replace('-', "/"))
        .unwrap_or_default()
}

/// classify_kind separates the user's interactive work from automated and
/// fixture runs, using the working directory first and the launch metadata as a
/// fallback.
pub fn classify_kind(cwd: &str, entrypoint: &str, session_kind: &str) -> Kind {
    if cwd.contains("/examples/") {
        Kind::Example
    } else if cwd.contains("/.gym/worktrees") {
        Kind::Gym
    } else if entrypoint == "sdk-cli" {
        Kind::Sdk
    } else if cwd.contains("/.claude/worktrees") {
        Kind::Worktree
    } else if session_kind == "bg" {
        Kind::Background
    } else {
        Kind::Main
    }
}

/// HeadScan is the static metadata recovered from a transcript head.
struct HeadScan {
    cwd: String,
    git_branch: String,
    version: String,
    entrypoint: String,
    session_kind: String,
    title: String,
    has_messages: bool,
}

/// scan_transcript_head reads up to MAX_META_LINES and fills cwd, branch,
/// version and title, returning the launch entrypoint and session kind for
/// classification. A custom title wins over an AI title, which wins over the
/// first user prompt; the last-seen title of each kind is kept so a renamed
/// session shows its current name.
fn scan_transcript_head(path: &Path) -> HeadScan {
    let mut scan = HeadScan {
        cwd: String::new(),
        git_branch: String::new(),
        version: String::new(),
        entrypoint: String::new(),
        session_kind: String::new(),
        title: String::new(),
        has_messages: false,
    };
    let f = match File::open(path) {
        Ok(f) => f,
        Err(_) => return scan,
    };
    let mut r = BufReader::with_capacity(64 * 1024, f);

    let mut custom_title = String::new();
    let mut ai_title = String::new();
    let mut first_prompt = String::new();

    for _ in 0..MAX_META_LINES {
        let (raw, too_long, eof) = read_bounded_line(&mut r, MAX_LINE_BYTES);
        if !raw.is_empty() && !too_long {
            if let Ok(line) = serde_json::from_slice::<TranscriptLine>(&raw) {
                if scan.cwd.is_empty() && !line.cwd.is_empty() {
                    scan.cwd = line.cwd.clone();
                }
                if scan.git_branch.is_empty() && !line.git_branch.is_empty() {
                    scan.git_branch = line.git_branch.clone();
                }
                if scan.version.is_empty() && !line.version.is_empty() {
                    scan.version = line.version.clone();
                }
                if scan.entrypoint.is_empty() && !line.entrypoint.is_empty() {
                    scan.entrypoint = line.entrypoint.clone();
                }
                if scan.session_kind.is_empty() && !line.session_kind.is_empty() {
                    scan.session_kind = line.session_kind.clone();
                }
                match line.r#type.as_str() {
                    "custom-title" => {
                        if !line.custom_title.is_empty() {
                            custom_title = line.custom_title.clone();
                        }
                    }
                    "ai-title" => {
                        if !line.ai_title.is_empty() {
                            ai_title = line.ai_title.clone();
                        }
                    }
                    "user" => {
                        scan.has_messages = true;
                        if first_prompt.is_empty() {
                            if let Some(msg) = &line.message {
                                if msg.role == "user" {
                                    first_prompt = clean_prompt(&first_user_text(&msg.content));
                                }
                            }
                        }
                    }
                    "assistant" => {
                        scan.has_messages = true;
                    }
                    _ => {}
                }
            }
        }
        if eof {
            break;
        }
    }
    scan.title = pick_title(&custom_title, &ai_title, &first_prompt);
    scan
}

/// read_bounded_line reads one newline-delimited line, keeping at most `max`
/// bytes. Lines longer than `max` (file snapshots, pasted images) are drained
/// and flagged via `too_long` so the caller skips them rather than stalling the
/// scan. The third return value is true once the file ends (EOF or read error).
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
        // read_until stops at the delimiter or EOF; n == 0 means EOF reached.
        if n == 0 || chunk.last() == Some(&b'\n') {
            trim_eol(&mut line);
            return (line, too_long, n == 0);
        }
    }
}

/// trim_eol strips a trailing `\r\n` / `\n` (matching Go's TrimRight).
fn trim_eol(line: &mut Vec<u8>) {
    while matches!(line.last(), Some(b'\n') | Some(b'\r')) {
        line.pop();
    }
}

/// read_last_message returns a cleaned preview of the most recent
/// user/assistant message, scanning only the transcript tail so it stays fast
/// on large files.
fn read_last_message(path: &Path, size: u64) -> String {
    let mut f = match File::open(path) {
        Ok(f) => f,
        Err(_) => return String::new(),
    };
    let offset = size.saturating_sub(LAST_MESSAGE_SCAN_BYTES);
    if f.seek(SeekFrom::Start(offset)).is_err() {
        return String::new();
    }
    let mut data = String::new();
    if f.read_to_string(&mut data).is_err() {
        // Tail may begin mid-UTF8 sequence; fall back to lossy bytes.
        let mut f2 = match File::open(path) {
            Ok(f) => f,
            Err(_) => return String::new(),
        };
        if f2.seek(SeekFrom::Start(offset)).is_err() {
            return String::new();
        }
        let mut bytes = Vec::new();
        if f2.read_to_end(&mut bytes).is_err() {
            return String::new();
        }
        data = String::from_utf8_lossy(&bytes).into_owned();
    }
    let mut lines: Vec<&str> = data.split('\n').collect();
    if offset > 0 && !lines.is_empty() {
        lines.remove(0); // drop the partial first line from mid-file seek
    }
    last_message_preview(&lines)
}

/// last_message_preview walks the lines from the end and returns the first
/// displayable user/assistant text, truncated to a preview length.
fn last_message_preview(lines: &[&str]) -> String {
    for line_str in lines.iter().rev() {
        let line: TranscriptLine = match serde_json::from_str(line_str) {
            Ok(l) => l,
            Err(_) => continue,
        };
        let msg = match &line.message {
            Some(m) => m,
            None => continue,
        };
        if line.r#type != "user" && line.r#type != "assistant" {
            continue;
        }
        // Keep the full multi-line message (wrappers stripped) so the preview can
        // expand beyond the first line.
        let text = strip_wrappers(&first_user_text(&msg.content));
        if !text.trim().is_empty() {
            return truncate(text.trim(), 600);
        }
    }
    String::new()
}

fn pick_title(custom: &str, ai: &str, prompt: &str) -> String {
    if !custom.is_empty() {
        custom.to_string()
    } else if !ai.is_empty() {
        ai.to_string()
    } else {
        truncate(prompt.trim(), 80)
    }
}

/// first_user_text extracts displayable text from a message's content, which is
/// either a plain string or an array of typed content blocks.
fn first_user_text(raw: &serde_json::Value) -> String {
    if raw.is_null() {
        return String::new();
    }
    if let Some(s) = raw.as_str() {
        return s.to_string();
    }
    if let Some(blocks) = raw.as_array() {
        for b in blocks {
            let ty = b.get("type").and_then(|v| v.as_str()).unwrap_or("");
            let text = b.get("text").and_then(|v| v.as_str()).unwrap_or("");
            if ty == "text" && !text.is_empty() {
                return text.to_string();
            }
        }
    }
    String::new()
}

/// strip_wrappers removes any leading run of harness-injected angle-bracket
/// blocks (system reminders, command caveats, slash-command envelopes) so what
/// remains is what the user/assistant actually wrote. Newlines are KEPT, so a
/// multi-line message stays multi-line (used for the expandable preview).
fn strip_wrappers(s: &str) -> String {
    let mut s = s.to_string();
    loop {
        s = s.trim().to_string();
        if !s.starts_with('<') {
            break;
        }
        let end = match s.find('>') {
            Some(e) => e,
            None => break,
        };
        let tag = &s[1..end];
        let name = tag.split(' ').next().unwrap_or("");
        if !name.is_empty() && !name.starts_with('/') {
            let close = format!("</{name}>");
            if let Some(idx) = s.find(&close) {
                s = s[idx + close.len()..].to_string();
                continue;
            }
        }
        s = s[end + 1..].to_string();
    }
    s
}

/// clean_prompt is strip_wrappers reduced to the first line — for the one-line
/// title fallback.
fn clean_prompt(s: &str) -> String {
    let s = strip_wrappers(s);
    match s.split_once('\n') {
        Some((line, _)) => line.to_string(),
        None => s,
    }
}

/// truncate caps a string at `n` chars (Unicode scalar values, matching the
/// preview lengths the UI expects), appending an ellipsis when trimmed. Go
/// counts bytes here; counting chars avoids splitting a multi-byte boundary
/// while staying visually equivalent for the ASCII-dominant prompts and the
/// 80/600-char limits in play.
fn truncate(s: &str, n: usize) -> String {
    if s.chars().count() <= n {
        return s.to_string();
    }
    let truncated: String = s.chars().take(n).collect();
    format!("{truncated}…")
}

/// mtime returns a file's modification time both as an RFC3339 UTC string (for
/// the API) and as nanoseconds since the Unix epoch (for sorting newest-first
/// without re-parsing).
fn mtime(info: &fs::Metadata) -> (String, i128) {
    let modified = match info.modified() {
        Ok(m) => m,
        Err(_) => return (String::new(), 0),
    };
    match modified.duration_since(std::time::UNIX_EPOCH) {
        Ok(d) => {
            let secs = d.as_secs() as i64;
            let nanos = d.subsec_nanos();
            let total_nanos = secs as i128 * 1_000_000_000 + nanos as i128;
            (crate::timefmt::rfc3339(secs, nanos), total_nanos)
        }
        Err(e) => {
            // Pre-epoch mtime: encode a negative instant for both fields.
            let d = e.duration();
            let secs = -(d.as_secs() as i64);
            let total_nanos = secs as i128 * 1_000_000_000 - d.subsec_nanos() as i128;
            (crate::timefmt::rfc3339(secs, 0), total_nanos)
        }
    }
}
