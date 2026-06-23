package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// liveSession is the subset of a ~/.claude/sessions/<pid>.json file this
// package reads. Claude Code writes one such file per running process.
type liveSession struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	Status    string `json:"status"` // "busy" while working, "idle" between turns
	UpdatedAt int64  `json:"updatedAt"`
}

// liveInfo is the resolved live state of a session keyed by sessionId.
type liveInfo struct {
	pid    int
	status Status
}

// resolveLiveStatuses reads the per-process session files and keeps only those
// whose process is actually alive, mapping sessionId to its live state. When
// several live files name the same session, the most recently updated wins.
func resolveLiveStatuses() map[string]liveInfo {
	out := map[string]liveInfo{}
	dir := sessionsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	best := map[string]int64{} // sessionId -> updatedAt of the chosen file
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		ls, ok := readLiveSession(filepath.Join(dir, e.Name()))
		if !ok || ls.SessionID == "" || !processAlive(ls.PID) {
			continue
		}
		if prev, seen := best[ls.SessionID]; seen && ls.UpdatedAt <= prev {
			continue
		}
		best[ls.SessionID] = ls.UpdatedAt
		out[ls.SessionID] = liveInfo{pid: ls.PID, status: classify(ls.Status)}
	}
	return out
}

func readLiveSession(path string) (liveSession, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return liveSession{}, false
	}
	var ls liveSession
	if json.Unmarshal(b, &ls) != nil {
		return liveSession{}, false
	}
	return ls, true
}

// classify maps Claude Code's reported process status to our Status. Anything
// that isn't actively working is treated as waiting on the user.
func classify(raw string) Status {
	if raw == "busy" {
		return StatusBusy
	}
	return StatusWaiting
}

// processAlive reports whether a PID names a running process. Signal 0 performs
// existence/permission checks without delivering a signal.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
