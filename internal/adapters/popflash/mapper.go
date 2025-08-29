package popflash

import (
	"fmt"
	"strings"
	"time"

	"github.com/jose-valero/popflash-queue-bot/internal/ui"
)

// Convierte el DTO apiMatch en el tipo que hoy consume la UI (ui.MatchCard).
func toUIMatchCard(m apiMatch) ui.MatchCard {
	t1, t2 := splitTeams(m)

	return ui.MatchCard{
		ID:      itoa(m.ID),
		Map:     safeStr(m.Map),
		Region:  dcName(m.Datacenter),
		Started: parseTime(m.CreatedAt),
		Team1:   t1,
		Team2:   t2,
		Score1:  m.Score1,
		Score2:  m.Score2,
	}
}

func splitTeams(m apiMatch) (t1, t2 []string) {
	for _, um := range m.Users {
		name := safeStr(um.User.Name)
		if name == "" || name == "—" {
			continue
		}
		if um.Team != nil && *um.Team == 2 {
			t2 = append(t2, name)
		} else {
			t1 = append(t1, name)
		}
	}
	return
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }

func safeStr(p *string) string {
	if p == nil {
		return "—"
	}
	s := strings.TrimSpace(*p)
	if s == "" || s == "-" {
		return "—"
	}
	return s
}

func parseTime(ts *string) time.Time {
	if ts == nil || *ts == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, *ts); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339Nano, *ts); err == nil {
		return t
	}
	return time.Time{}
}
