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
func RenderQueuesEmbed(qs []*queue.Queue) *discordgo.MessageEmbed {
	if len(qs) == 0 {
		return &discordgo.MessageEmbed{
			Title:       "❌ No queues",
			Description: "Use `/startqueue` to create the first one.",
			Color:       0xED4245,
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
		Color:       0xB069FF,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "XCG BOT • don't lose your spot",
		},
	}
}
