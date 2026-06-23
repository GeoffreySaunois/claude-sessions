package core

import "sort"

// LoadSessions returns every discovered session, enriched with live status and
// the user's organization metadata, sorted most-recently-active first.
func LoadSessions() ([]Session, error) {
	sessions, err := discoverTranscripts()
	if err != nil {
		return nil, err
	}
	live := resolveLiveStatuses()
	store, err := LoadMetaStore()
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		if info, ok := live[sessions[i].ID]; ok {
			sessions[i].Status = info.status
			sessions[i].PID = info.pid
		}
		store.apply(&sessions[i])
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastActive.After(sessions[j].LastActive)
	})
	return sessions, nil
}
