package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

type MatchCard struct {
	ID      string
	Map     string
	Region  string
	Started time.Time
	Team1   []string
	Team2   []string
	// NUEVO: marcador en vivo (usa -1 si no disponible)
	Score1 *int
	Score2 *int
}

/* ---------- helpers ---------- */

// —— formatea el marcador “6–7”; si no hay datos, devuelve “—”
func scoreLine(c MatchCard) string {
	s := func(p *int) string {
		if p == nil {
			return "—"
		}
		return strconv.Itoa(*p)
	}
	return fmt.Sprintf("**Marcador:** %s–%s", s(c.Score1), s(c.Score2))
}

func list(names []string) string {
	if len(names) == 0 {
		return "—"
	}
	var b strings.Builder
	for _, n := range names {
		fmt.Fprintf(&b, "• %s\n", n)
	}
	return b.String()
}

func queueTitle(isOpen bool) string {
	state := "🔓 Cola abierta"
	if !isOpen {
		state = "🔒 Cola cerrada"
	}
	return fmt.Sprintf("Ellos la llevan — Fila Global • %s", state)
}

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

func scoreSuffix(c MatchCard) string {
	if c.Score1 == nil || c.Score2 == nil {
		return ""
	}
	return fmt.Sprintf(" • **%d–%d**", *c.Score1, *c.Score2)
}

// Tarjeta compacta de una partida (columna inline)
func matchField(c MatchCard) *discordgo.MessageEmbedField {
	name := fmt.Sprintf("#%s • %s • @%s • ⏱ %s%s",
		safe(c.ID), safe(c.Map), safe(c.Region), humanSince(c.Started), scoreSuffix(c))

	team1 := bulletList(c.Team1, 5)
	team2 := bulletList(c.Team2, 5)

	// 1) línea de score justo debajo del “name”
	// 2) luego el bloque con los equipos (blockquote para la barra gris)
	val := scoreLine(c) + "\n\n" + quoteBlock(
		fmt.Sprintf("**Team #1**\n%s\n\n**Team #2**\n%s", team1, team2),
	)

	// Agrega una línea “casi vacía” para crear aire entre tarjetas
	val = val + "\n\u200B"

	return &discordgo.MessageEmbedField{
		Name:   name,
		Value:  val,
		Inline: true, // ← dos columnas
	}
}

/* ---------- embed principal (uno solo) ---------- */

func RenderQueuesEmbed(qs []*queue.Queue, isOpen bool, cards []MatchCard) *discordgo.MessageEmbed {
	color := map[bool]int{true: 0x57F287, false: 0x808080}[isOpen]

	emb := &discordgo.MessageEmbed{
		Title:       queueTitle(isOpen),
		Description: buildQueuesDescription(qs),
		Color:       color,
	}

	// Bloque “Partidas activas”
	emb.Fields = append(emb.Fields, &discordgo.MessageEmbedField{
		Name:   "Partidas activas",
		Value:  "\u200B",
		Inline: false,
	})

	if len(cards) == 0 {
		emb.Fields = append(emb.Fields, &discordgo.MessageEmbedField{
			Name:   "\u200B",
			Value:  "_Ninguna en curso_",
			Inline: false,
		})
		return emb
	}

	// Máximo 2 partidas visibles, lado a lado
	limit := 2
	if len(cards) < limit {
		limit = len(cards)
	}
	for i := 0; i < limit; i++ {
		emb.Fields = append(emb.Fields, matchField(cards[i]))
	}
	// Espaciador de fila completa para más aire entre bloques
	emb.Fields = append(emb.Fields, &discordgo.MessageEmbedField{Name: "\u200B", Value: "\u200B", Inline: false})

	return emb
}

func buildQueuesDescription(qs []*queue.Queue) string {
	if len(qs) == 0 {
		return "Use `/startqueue` para crear la primera."
	}
	var b strings.Builder
	for idx, q := range qs {
		fmt.Fprintf(&b, "**Fila #%d** (%d/%d)\n", idx+1, len(q.Players), q.Capacity)
		if len(q.Players) == 0 {
			b.WriteString("_(empty)_\n\n")
			continue
		}
		for i, p := range q.Players {
			fmt.Fprintf(&b, "%d) %s\n", i+1, p.Username)
		}
		b.WriteString("\n")
	}
	return b.String()
}
