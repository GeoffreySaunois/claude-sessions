// Command ccs is a temporary harness to verify the core package against real
// on-disk sessions. The TUI and web frontends will replace it.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"claude-sessions/core"
)

func main() {
	sessions, err := core.LoadSessions()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(os.Args) > 1 && os.Args[1] == "--json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(sessions)
		return
	}
	counts := map[core.Status]int{}
	for _, s := range sessions {
		counts[s.Status]++
	}
	fmt.Printf("total=%d busy=%d waiting=%d inactive=%d\n",
		len(sessions), counts[core.StatusBusy], counts[core.StatusWaiting], counts[core.StatusInactive])
	for i, s := range sessions {
		if i >= 15 {
			break
		}
		fmt.Printf("%-9s %-22s %-40.40s %s\n", s.Status, projBase(s.Cwd), s.Title, s.LastActive.Format("01-02 15:04"))
	}
}

func projBase(cwd string) string {
	for i := len(cwd) - 1; i >= 0; i-- {
		if cwd[i] == '/' {
			return cwd[i+1:]
		}
	}
	return cwd
}
