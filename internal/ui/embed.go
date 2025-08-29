package ui

import (
	"fmt"
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
	Score1  *int
	Score2  *int
}

func buildQueuesDescription(qs []*queue.Queue) string {
	if len(qs) == 0 {
		return "Use `/startqueue` para crear la primera."
	}
	var b strings.Builder
	for idx, q := range qs {
		fmt.Fprintf(&b, "**Fila #%d** (%d/%d)\n", idx+1, len(q.Players), q.Capacity) // if u need it changes for ur language, "fila" means "queue"
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

// compact card of a match (inline column)
func matchField(c MatchCard) *discordgo.MessageEmbedField {
	// match info
	name := fmt.Sprintf("#%s • %s • @%s • ⏱ %s",
		safe(c.ID), safe(c.Map), safe(c.Region), humanSince(c.Started))

	team1 := bulletList(c.Team1, 5)
	team2 := bulletList(c.Team2, 5)

	// 1) score line
	// 2) match info block (players vs players)
	val := scoreLine(c) + "\n\n" + quoteBlock(
		fmt.Sprintf("**Team #1**\n%s\n\n**Team #2**\n%s", team1, team2),
	)

	// empty line between matches
	val = val + "\n\u200B"

	return &discordgo.MessageEmbedField{
		Name:   name,
		Value:  val,
		Inline: true, // ← 2 columns
	}
}

// ---------- principal embed (just one) ----------
func RenderQueuesEmbed(qs []*queue.Queue, isOpen bool, cards []MatchCard) *discordgo.MessageEmbed {
	color := map[bool]int{true: 0x57F287, false: 0x808080}[isOpen]

	emb := &discordgo.MessageEmbed{
		Title:       queueTitle(isOpen),
		Description: buildQueuesDescription(qs),
		Color:       color,
	}

	// block of active matches
	emb.Fields = append(emb.Fields, &discordgo.MessageEmbedField{
		Name:   "Partidas activas", // if u need it changes for ur language, "Partidas activas" means "active matches"
		Value:  "\u200B",
		Inline: false,
	})

	if len(cards) == 0 {
		emb.Fields = append(emb.Fields, &discordgo.MessageEmbedField{
			Name:   "\u200B",
			Value:  "_Ninguna en curso_", // if u need it changes for ur language, "ninguna en curso" means "none in progress"
			Inline: false,
		})
		return emb
	}

	// max 2 matches
	limit := 2
	if len(cards) < limit {
		limit = len(cards)
	}
	for i := 0; i < limit; i++ {
		emb.Fields = append(emb.Fields, matchField(cards[i]))
	}
	// spacer of the raw row
	emb.Fields = append(emb.Fields, &discordgo.MessageEmbedField{Name: "\u200B", Value: "\u200B", Inline: false})

	return emb
}
