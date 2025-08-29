package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func scoreLine(c MatchCard) string {
	s := func(p *int) string {
		if p == nil {
			return "—"
		}
		return strconv.Itoa(*p)
	}
	return fmt.Sprintf("**score:** %s–%s", s(c.Score1), s(c.Score2))
}

func queueTitle(isOpen bool) string {
	state := "🔓 Cola abierta"
	if !isOpen {
		state = "🔒 Cola cerrada"
	}
	return fmt.Sprintf("Ellos la llevan — Fila Global - %s", state)
}

// humanize the time of match
func humanSince(t time.Time) string {
	if t.IsZero() {
		return "desconocido"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "hace segundos"
	}
	if d < time.Hour {
		return fmt.Sprintf("hace %d min", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("hace %dh", h)
	}
	return fmt.Sprintf("hace %dh %dm", h, m)
}

// fallback to falsy data
func safe(s string) string {
	t := strings.TrimSpace(s)
	if t == "" || t == "-" {
		return "—"
	}
	return t
}

func bulletList(team []string, max int) string {
	if len(team) == 0 {
		return "—"
	}
	if max > 0 && len(team) > max {
		team = team[:max]
	}
	var b strings.Builder
	for _, p := range team {
		fmt.Fprintf(&b, "• %s\n", p)
	}
	return strings.TrimRight(b.String(), "\n")
}

func quoteBlock(s string) string {
	if s == "" {
		return "> —"
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = "> " + lines[i]
	}
	return strings.Join(lines, "\n")
}
