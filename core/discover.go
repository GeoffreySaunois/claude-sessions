package core

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// metadataScanBytes caps how much of each transcript is parsed for metadata.
// cwd/branch appear in the first turn and titles recur on nearly every turn,
// so the head reliably carries them while keeping a full-corpus scan fast even
// across thousands of multi-megabyte transcripts.
const metadataScanBytes = 512 * 1024

// transcriptLine is the subset of a transcript JSONL row this package reads.
type transcriptLine struct {
	Type        string         `json:"type"`
	Cwd         string         `json:"cwd"`
	GitBranch   string         `json:"gitBranch"`
	Version     string         `json:"version"`
	Entrypoint  string         `json:"entrypoint"`  // "cli", "sdk-cli", …
	SessionKind string         `json:"sessionKind"` // "bg", …
	AiTitle     string         `json:"aiTitle"`     // on ai-title rows
	CustomTitle string         `json:"customTitle"` // on custom-title rows
	Message     *transcriptMsg `json:"message"`
}

type transcriptMsg struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// discoverTranscripts returns every session transcript across all projects,
// parsed for static metadata. Status and sidecar fields are filled in later.
func discoverTranscripts() ([]Session, error) {
	root := projectsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var sessions []Session
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		projDir := filepath.Join(root, e.Name())
		files, err := os.ReadDir(projDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			s, ok := parseTranscript(projDir, f.Name())
			if ok {
				sessions = append(sessions, s)
			}
		}
	}
	return sessions, nil
}

// parseTranscript reads one transcript's head and extracts session metadata.
func parseTranscript(projDir, name string) (Session, bool) {
	path := filepath.Join(projDir, name)
	info, err := os.Stat(path)
	if err != nil {
		return Session{}, false
	}
	s := Session{
		ID:         strings.TrimSuffix(name, ".jsonl"),
		Path:       path,
		ProjectDir: projDir,
		LastActive: info.ModTime(),
		SizeBytes:  info.Size(),
		Status:     StatusInactive,
	}
	entrypoint, sessionKind := scanTranscriptHead(path, &s)
	s.Kind = classifyKind(s.Cwd, entrypoint, sessionKind)
	s.LastMessage = readLastMessage(path, info.Size())
	if s.Title == "" {
		s.Title = s.ID
	}
	return s, true
}

// classifyKind separates the user's interactive work from automated and fixture
// runs, using the working directory first and the launch metadata as a fallback.
func classifyKind(cwd, entrypoint, sessionKind string) Kind {
	switch {
	case strings.Contains(cwd, "/examples/"):
		return KindExample
	case strings.Contains(cwd, "/.gym/worktrees"):
		return KindGym
	case entrypoint == "sdk-cli":
		return KindSDK
	case strings.Contains(cwd, "/.claude/worktrees"):
		return KindWorktree
	case sessionKind == "bg":
		return KindBackground
	default:
		return KindMain
	}
}

// scanTranscriptHead reads up to metadataScanBytes and fills cwd, branch,
// version and title, returning the launch entrypoint and session kind for
// classification. A custom title wins over an ai title, which wins over the
// first user prompt; the last-seen title of each kind is kept so a renamed
// session shows its current name.
func scanTranscriptHead(path string, s *Session) (entrypoint, sessionKind string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	var customTitle, aiTitle, firstPrompt string
	r := bufio.NewReader(io.LimitReader(f, metadataScanBytes))
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var line transcriptLine
		if json.Unmarshal(sc.Bytes(), &line) != nil {
			continue
		}
		if s.Cwd == "" && line.Cwd != "" {
			s.Cwd = line.Cwd
		}
		if s.GitBranch == "" && line.GitBranch != "" {
			s.GitBranch = line.GitBranch
		}
		if s.Version == "" && line.Version != "" {
			s.Version = line.Version
		}
		if entrypoint == "" && line.Entrypoint != "" {
			entrypoint = line.Entrypoint
		}
		if sessionKind == "" && line.SessionKind != "" {
			sessionKind = line.SessionKind
		}
		switch line.Type {
		case "custom-title":
			if line.CustomTitle != "" {
				customTitle = line.CustomTitle
			}
		case "ai-title":
			if line.AiTitle != "" {
				aiTitle = line.AiTitle
			}
		case "user":
			if firstPrompt == "" && line.Message != nil && line.Message.Role == "user" {
				firstPrompt = cleanPrompt(firstUserText(line.Message.Content))
			}
		}
	}
	s.Title = pickTitle(customTitle, aiTitle, firstPrompt)
	return entrypoint, sessionKind
}

// lastMessageScanBytes caps how much of the transcript tail is read for the
// last-message preview.
const lastMessageScanBytes = 128 * 1024

// readLastMessage returns a cleaned preview of the most recent user/assistant
// message, scanning only the transcript tail so it stays fast on large files.
func readLastMessage(path string, size int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	offset := size - lastMessageScanBytes
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return ""
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if offset > 0 && len(lines) > 0 {
		lines = lines[1:] // drop the partial first line from mid-file seek
	}
	return lastMessagePreview(lines)
}

// lastMessagePreview walks the lines from the end and returns the first
// displayable user/assistant text, truncated to a preview length.
func lastMessagePreview(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		var line transcriptLine
		if json.Unmarshal([]byte(lines[i]), &line) != nil || line.Message == nil {
			continue
		}
		if line.Type != "user" && line.Type != "assistant" {
			continue
		}
		text := cleanPrompt(firstUserText(line.Message.Content))
		if strings.TrimSpace(text) != "" {
			return truncate(strings.TrimSpace(text), 600)
		}
	}
	return ""
}

func pickTitle(custom, ai, prompt string) string {
	switch {
	case custom != "":
		return custom
	case ai != "":
		return ai
	default:
		return truncate(strings.TrimSpace(prompt), 80)
	}
}

// firstUserText extracts displayable text from a user message's content, which
// is either a plain string or an array of typed content blocks.
func firstUserText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}

// cleanPrompt strips harness-injected wrappers (system reminders, command
// caveats, slash-command envelopes) so a prompt-derived title shows what the
// user actually typed rather than machinery. It removes any leading run of
// angle-bracket-tagged blocks and returns the first real line of prose.
func cleanPrompt(s string) string {
	for {
		s = strings.TrimSpace(s)
		if !strings.HasPrefix(s, "<") {
			break
		}
		end := strings.Index(s, ">")
		if end < 0 {
			break
		}
		tag := s[1:end]
		if name, _, _ := strings.Cut(tag, " "); name != "" && !strings.HasPrefix(name, "/") {
			if close := strings.Index(s, "</"+name+">"); close >= 0 {
				s = s[close+len("</"+name+">"):]
				continue
			}
		}
		s = s[end+1:]
	}
	if line, _, ok := strings.Cut(s, "\n"); ok {
		return line
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
