package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

// Embed con TODAS las colas del canal
func renderQueuesEmbed(qs []*queue.Queue) *discordgo.MessageEmbed {
	if len(qs) == 0 {
		return &discordgo.MessageEmbed{
			Title:       "❌ No hay colas activas",
			Description: "Usa `/startqueue` para crear la primera.",
			Color:       0xED4245,
		}
	}

	var b strings.Builder
	for idx, q := range qs {
		fmt.Fprintf(&b, "**Banca #%d** (%d/%d)\n", idx+1, len(q.Players), q.Capacity)
		if len(q.Players) == 0 {
			b.WriteString("_(vacía)_\n\n")
			continue
		}
		for i, p := range q.Players {
			fmt.Fprintf(&b, "%d) %s\n", i+1, p.Username)
		}
		b.WriteString("\n")
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Ellos la llevan — %d cola(s)", len(qs)),
		Description: b.String(),
		Color:       0xB069FF,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "XCG BOT • no pierdas tu lugar",
		},
	}
}

// Componentes para multi-cola:
//   - Fila base: Join / Leave
//   - Select: acciones por cola (Reset / Close) en la cola seleccionada
func componentsForQueues(qs []*queue.Queue) []discordgo.MessageComponent {
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "La Llevo", Style: discordgo.PrimaryButton, CustomID: "queue_join", Emoji: &discordgo.ComponentEmoji{Name: "🌕"}},
				discordgo.Button{Label: "Chau", Style: discordgo.SecondaryButton, CustomID: "queue_leave", Emoji: &discordgo.ComponentEmoji{Name: "👋"}},
			},
		},
	}

	if len(qs) > 0 {
		opts := make([]discordgo.SelectMenuOption, 0, len(qs)*2)
		for idx := range qs {
			n := idx + 1
			opts = append(opts,
				discordgo.SelectMenuOption{
					Label:       fmt.Sprintf("Reset Q#%d", n),
					Value:       fmt.Sprintf("reset:%d", n),
					Description: "Vacía esa cola",
				},
				discordgo.SelectMenuOption{
					Label:       fmt.Sprintf("Close Q#%d", n),
					Value:       fmt.Sprintf("close:%d", n),
					Description: "Elimina esa cola",
				},
			)
		}
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "queue_action",
					Placeholder: "Acciones por cola… (reset/close)",
					Options:     opts,
				},
			},
		})
	}
	// NUEVO: Select para kickear jugadores (máx 25)
	if hasAnyPlayers(qs) {
		kopts := kickOptions(qs)
		if len(kopts) > 0 {
			components = append(components, discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    "queue_kick",
						Placeholder: "Kickear jugador…",
						Options:     kopts,
					},
				},
			})
		}
	}

	return components
}

// ¿hay al menos un jugador en alguna cola?
func hasAnyPlayers(qs []*queue.Queue) bool {
	for _, q := range qs {
		if len(q.Players) > 0 {
			return true
		}
	}
	return false
}

// Construye opciones para el select "kick", tope 25 (límite de Discord)
func kickOptions(qs []*queue.Queue) []discordgo.SelectMenuOption {
	opts := make([]discordgo.SelectMenuOption, 0, 25)
	for qi, q := range qs {
		for _, p := range q.Players {
			opts = append(opts, discordgo.SelectMenuOption{
				Label:       fmt.Sprintf("Kick %s (Q#%d)", p.Username, qi+1),
				Value:       "uid:" + p.ID, // el value viaja al handler
				Description: "",            // opcional
			})
			if len(opts) == 25 {
				return opts
			}
		}
	}
	return opts
}
