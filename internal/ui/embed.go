// internal/ui/embed.go
// Pure-ish UI helpers for building the main queue embed.

package ui

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

// RenderQueuesEmbed builds a single embed showing all queues in a channel.
func RenderQueuesEmbed(qs []*queue.Queue, open bool) *discordgo.MessageEmbed {
	footer := &discordgo.MessageEmbedFooter{
		Text: map[bool]string{
			true:  "🔓 Cola abierta — no pierdas tu slot",
			false: "🔒 Cola cerrada — espera la próxima partida",
		}[open],
	}

	// Si no querés cambiar color, dejá ambos en 0xB069FF.
	color := map[bool]int{
		true:  0x57F287, // verde cuando abierta
		false: 0x808080, // gris cuando cerrada
	}[open]

	if len(qs) == 0 {
		return &discordgo.MessageEmbed{
			Title:       "❌ No queues",
			Description: "Use `/startqueue` to create the first one.",
			Color:       color,
			Footer:      footer,
		}
	}

	var b strings.Builder
	for idx, q := range qs {
		fmt.Fprintf(&b, "**Queue #%d** (%d/%d)\n", idx+1, len(q.Players), q.Capacity)
		if len(q.Players) == 0 {
			b.WriteString("_(empty)_\n\n")
			continue
		}
		for i, p := range q.Players {
			fmt.Fprintf(&b, "%d) %s\n", i+1, p.Username)
		}
		b.WriteString("\n")
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Ellos la llevan — %d queue(s)", len(qs)),
		Description: b.String(),
		Color:       color,
		Footer:      footer,
	}
}
