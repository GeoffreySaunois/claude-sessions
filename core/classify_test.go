package core

import "testing"

// TestClassifyKind pins the precedence: a fixture/worktree path wins over the
// launch metadata, so a gym example session never reads as a plain "main".
func TestClassifyKind(t *testing.T) {
	cases := []struct {
		cwd, entrypoint, sessionKind string
		want                         Kind
	}{
		{"/Users/x/Projects/Swaap/Main", "cli", "", KindMain},
		{"/x/src/python/gym/examples/_hooktest-longscript", "cli", "", KindExample},
		{"/x/.gym/worktrees/reporter/src", "cli", "", KindGym},
		{"/x/proj", "sdk-cli", "", KindSDK},
		{"/x/.claude/worktrees/feat-foo", "cli", "", KindWorktree},
		{"/x/proj", "cli", "bg", KindBackground},
		// path beats metadata: an examples/ dir launched via sdk is still example
		{"/x/examples/demo", "sdk-cli", "", KindExample},
	}
	for _, c := range cases {
		if got := classifyKind(c.cwd, c.entrypoint, c.sessionKind); got != c.want {
			t.Errorf("classifyKind(%q,%q,%q) = %q, want %q", c.cwd, c.entrypoint, c.sessionKind, got, c.want)
		}
	}
}

// TestInsertSorted guards dedup, ordering, and the empty-name no-op the option
// universe relies on.
func TestInsertSorted(t *testing.T) {
	xs := insertSorted(nil, "work")
	xs = insertSorted(xs, "admin")
	xs = insertSorted(xs, "work") // duplicate ignored
	xs = insertSorted(xs, "")     // empty ignored
	want := []string{"admin", "work"}
	if len(xs) != len(want) {
		t.Fatalf("got %v, want %v", xs, want)
	}
	for i := range want {
		if xs[i] != want[i] {
			t.Fatalf("got %v, want %v", xs, want)
		}
	}
}

// TestLastMessagePreview takes the most recent displayable user/assistant text,
// skipping trailing non-message rows, and cleans harness wrappers.
func TestLastMessagePreview(t *testing.T) {
	lines := []string{
		`{"type":"user","message":{"role":"user","content":"first question"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"the final answer"}]}}`,
		`{"type":"ai-title","aiTitle":"some title"}`, // not a message -> skipped
	}
	if got := lastMessagePreview(lines); got != "the final answer" {
		t.Fatalf("preview = %q, want %q", got, "the final answer")
	}
}
