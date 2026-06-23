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
	scanTranscriptHead(path, &s)
	if s.Title == "" {
		s.Title = s.ID
	}
	return s, true
}

// scanTranscriptHead reads up to metadataScanBytes and fills cwd, branch,
// version and title. A custom title wins over an ai title, which wins over the
// first user prompt; the last-seen title of each kind is kept so a renamed
// session shows its current name.
func scanTranscriptHead(path string, s *Session) {
	f, err := os.Open(path)
	if err != nil {
		return
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
