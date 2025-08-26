// internal/ui/components.go
package ui

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

// ---------- PÚBLICO: solo Join/Leave + botón "Admin…" ----------

func ComponentsForQueues(qs []*queue.Queue, isOpen bool) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "La Llevo",
					Style:    discordgo.PrimaryButton,
					CustomID: "queue_join",
					Emoji:    &discordgo.ComponentEmoji{Name: "🌕"},
					Disabled: !isOpen, // si la cola está cerrada, deshabilita el join
				},
				discordgo.Button{
					Label:    "Chau",
					Style:    discordgo.SecondaryButton,
					CustomID: "queue_leave",
					Emoji:    &discordgo.ComponentEmoji{Name: "👋"},
				},
				// visible para todos; solo los admins verán/usaràn el panel efímero
				discordgo.Button{
					Label:    "Admin…",
					Style:    discordgo.SecondaryButton,
					CustomID: "admin_panel",
					Emoji:    &discordgo.ComponentEmoji{Name: "👮"},
				},
			},
		},
	}
}

// ---------- ADMIN (EFÍMERO): solo selects (acciones + kick) ----------

func AdminComponentsForQueues(qs []*queue.Queue) []discordgo.MessageComponent {
	comps := make([]discordgo.MessageComponent, 0, 2)

	// Select de acciones por cola (Reset / Close). Cap a 12 colas (24 opciones).
	if len(qs) > 0 {
		n := len(qs)
		if n > 12 {
			n = 12
		}
		opts := make([]discordgo.SelectMenuOption, 0, n*2)
		for idx := 0; idx < n; idx++ {
			k := idx + 1
			opts = append(opts,
				discordgo.SelectMenuOption{
					Label:       fmt.Sprintf("Reset Q#%d", k),
					Value:       fmt.Sprintf("reset:%d", k),
					Description: "Clear that queue",
				},
				discordgo.SelectMenuOption{
					Label:       fmt.Sprintf("Close Q#%d", k),
					Value:       fmt.Sprintf("close:%d", k),
					Description: "Delete that queue",
				},
			)
		}
		comps = append(comps, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "queue_action",
					Placeholder: "Actions… (reset/close)",
					Options:     opts,
				},
			},
		})
	}

	// Select de Kick (hasta 25 players)
	kopts := kickOptions(qs)
	if len(kopts) > 0 {
		comps = append(comps, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "queue_kick",
					Placeholder: "Kick a player…",
					Options:     kopts,
				},
			},
		})
	}

	return comps
}

// ---------- helpers ----------

func hasAnyPlayers(qs []*queue.Queue) bool {
	for _, q := range qs {
		if len(q.Players) > 0 {
			return true
		}
	}
	return false
}

func kickOptions(qs []*queue.Queue) []discordgo.SelectMenuOption {
	opts := make([]discordgo.SelectMenuOption, 0, 25)
	for qi, q := range qs {
		for _, p := range q.Players {
			opts = append(opts, discordgo.SelectMenuOption{
				Label: fmt.Sprintf("Kick %s (Q#%d)", p.Username, qi+1),
				Value: "uid:" + p.ID,
			})
			if len(opts) == 25 {
				return opts
			}
		}
	}
	return opts
}
