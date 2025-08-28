package app

import (
	"sort"
	"sync"

	"github.com/jose-valero/popflash-queue-bot/internal/ui"
)

var (
	activeMu   sync.RWMutex
	activeByID = map[string]ui.MatchCard{} // id -> card
)

func ActivePut(card ui.MatchCard) {
	activeMu.Lock()
	activeByID[card.ID] = card
	activeMu.Unlock()
}

func ActiveUpdateScore(id string, s1, s2 *int) {
	activeMu.Lock()
	if c, ok := activeByID[id]; ok {
		c.Score1, c.Score2 = s1, s2
		activeByID[id] = c
	}
	activeMu.Unlock()
}

func ActiveRemove(id string) {
	activeMu.Lock()
	delete(activeByID, id)
	activeMu.Unlock()
}

func ActiveCount() int {
	activeMu.RLock()
	n := len(activeByID)
	activeMu.RUnlock()
	return n
}

func ActiveList() []ui.MatchCard {
	activeMu.RLock()
	out := make([]ui.MatchCard, 0, len(activeByID))
	for _, c := range activeByID {
		out = append(out, c)
	}
	activeMu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		ti, tj := out[i].Started, out[j].Started
		if ti.IsZero() {
			return false
		}
		if tj.IsZero() {
			return true
		}
		return ti.Before(tj)
	})
	return out
}
