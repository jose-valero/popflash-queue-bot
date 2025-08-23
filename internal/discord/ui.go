package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

func renderQueue(q *queue.Queue) string {
	if q == nil {
		return "❌ No active queue."
	}
	if len(q.Players) == 0 {
		return fmt.Sprintf("📋 **%s** (%d/%d)\n_(No hay nadie llevandola)_", q.Name, 0, q.Capacity)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "📋 **%s** (%d/%d)\n", q.Name, len(q.Players), q.Capacity)
	for idx, p := range q.Players {
		fmt.Fprintf(&b, "%d) %s\n", idx+1, p.Username)
	}
	return b.String()
}

// NUEVO: misma fila pero con Join deshabilitado si la cola está llena
func componentsRowDisabled(disabled bool) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "La Llevo", Style: discordgo.PrimaryButton, CustomID: "queue_join", Disabled: disabled},
				discordgo.Button{Label: "Chau", Style: discordgo.SecondaryButton, CustomID: "queue_leave"},
				discordgo.Button{Label: "Close", Style: discordgo.DangerButton, CustomID: "queue_close"},
				discordgo.Button{Label: "Reset", Style: discordgo.DangerButton, CustomID: "queue_reset"},
			},
		},
	}
}

func componentsForQueue(q *queue.Queue) []discordgo.MessageComponent {
	if q == nil {
		return nil
	}
	full := len(q.Players) >= q.Capacity
	return componentsRowDisabled(full)
}
