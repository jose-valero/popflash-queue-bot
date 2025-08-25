// internal/ui/components.go
// Build Discord components (buttons/select-menus) for the queues UI.

package ui

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

// ComponentsForQueues returns rows for:
//   - Row 1: Join / Leave
//   - Row 2: Queue actions (Reset/Close per queue) — optional when there are queues
//   - Row 3: Kick select (up to 25 players) — optional when there are players
func ComponentsForQueues(qs []*queue.Queue) []discordgo.MessageComponent {
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "La Llevo",
					Style:    discordgo.PrimaryButton,
					CustomID: "queue_join",
					Emoji:    &discordgo.ComponentEmoji{Name: "🌕"},
				},
				discordgo.Button{
					Label:    "Chau",
					Style:    discordgo.SecondaryButton,
					CustomID: "queue_leave",
					Emoji:    &discordgo.ComponentEmoji{Name: "👋"},
				},
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
					Description: "Clear that queue",
				},
				discordgo.SelectMenuOption{
					Label:       fmt.Sprintf("Close Q#%d", n),
					Value:       fmt.Sprintf("close:%d", n),
					Description: "Delete that queue",
				},
			)
		}
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "queue_action",
					Placeholder: "Actions… (reset/close)",
					Options:     opts,
				},
			},
		})
	}

	// Kick select (admins only at runtime; UI always rendered for now)
	if hasAnyPlayers(qs) {
		kopts := kickOptions(qs)
		if len(kopts) > 0 {
			components = append(components, discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    "queue_kick",
						Placeholder: "Kick a player…",
						Options:     kopts,
					},
				},
			})
		}
	}

	return components
}

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
