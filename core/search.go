package core

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// snippetContext is how many characters of surrounding text a search snippet
// keeps on each side of the first matched term.
const snippetContext = 90

// SearchTranscripts scans every transcript's conversation text and returns, for
// each session whose messages contain all whitespace-separated query terms
// (case-insensitive, AND), a short snippet around the first match. Files are
// scanned concurrently; oversized lines (snapshots, pasted images) are skipped.
func SearchTranscripts(query string) (map[string]string, error) {
	terms := strings.Fields(strings.ToLower(query))
	out := map[string]string{}
	if len(terms) == 0 {
		return out, nil
	}
	files, err := listTranscriptFiles()
	if err != nil {
		return nil, err
	}

	type hit struct {
		id, snippet string
	}
	hits := make(chan hit, 64)
	sem := make(chan struct{}, max(2, runtime.NumCPU()))
	var wg sync.WaitGroup
	for _, f := range files {
		wg.Add(1)
		go func(id, path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if snip, ok := searchFile(path, terms); ok {
				hits <- hit{id: id, snippet: snip}
			}
		}(f.id, f.path)
	}
	go func() { wg.Wait(); close(hits) }()

	for h := range hits {
		out[h.id] = h.snippet
	}
	return out, nil
}

type fileRef struct{ id, path string }

// listTranscriptFiles enumerates every session transcript across all projects.
func listTranscriptFiles() ([]fileRef, error) {
	root := projectsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var files []fileRef
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		fs, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range fs {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".jsonl") {
				files = append(files, fileRef{
					id:   strings.TrimSuffix(f.Name(), ".jsonl"),
					path: filepath.Join(dir, f.Name()),
				})
			}
		}
	}
	return files, nil
}

// searchFile assembles a transcript's user/assistant message text and reports a
// snippet if every term is present.
func searchFile(path string, terms []string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 64*1024)
	var b strings.Builder
	for {
		raw, tooLong, rerr := readBoundedLine(r, maxLineBytes)
		if len(raw) > 0 && !tooLong {
			var line transcriptLine
			if json.Unmarshal(raw, &line) == nil && line.Message != nil &&
				(line.Type == "user" || line.Type == "assistant") {
				if t := allMessageText(line.Message.Content); t != "" {
					b.WriteString(t)
					b.WriteByte('\n')
				}
			}
		}
		if rerr != nil {
			break
		}
	}

	text := b.String()
	lowered := strings.ToLower(text)
	for _, term := range terms {
		if !strings.Contains(lowered, term) {
			return "", false
		}
	}
	return snippetAround(text, lowered, terms[0]), true
}

// allMessageText joins every text block of a message's content, which is either
// a plain string or an array of typed blocks.
func allMessageText(raw json.RawMessage) string {
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
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, " ")
}

// snippetAround returns a single-line excerpt centered on the first occurrence
// of term, with ellipses where the text is trimmed.
func snippetAround(text, lowered, term string) string {
	i := strings.Index(lowered, term)
	if i < 0 {
		return truncate(collapseSpaces(text), 180)
	}
	start := i - snippetContext
	if start < 0 {
		start = 0
	}
	end := i + len(term) + snippetContext
	if end > len(text) {
		end = len(text)
	}
	snip := collapseSpaces(strings.ToValidUTF8(text[start:end], ""))
	if start > 0 {
		snip = "…" + snip
	}
	if end < len(text) {
		snip += "…"
	}
	return snip
}

// collapseSpaces folds any run of whitespace into a single space so a snippet
// drawn from a multi-line message renders on one line.
func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
