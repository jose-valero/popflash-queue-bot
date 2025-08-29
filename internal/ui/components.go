package ui

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

// bot bottoms, "queue" | "leave" | "admin"
func ComponentsForQueues(qs []*queue.Queue, isOpen bool) []discordgo.MessageComponent {
	joinID := "queue_join:open"
	if !isOpen {
		joinID = "queue_join:closed"
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "La Llevo", // if u need it changes for ur language, "la llevo" means "join"
					Style:    discordgo.PrimaryButton,
					CustomID: joinID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🌕"},
					Disabled: !isOpen, // disabled if the queue is closed
				},
				discordgo.Button{
					Label:    "Chau", // if u need it changes for ur language, "chau" means leave
					Style:    discordgo.SecondaryButton,
					CustomID: "queue_leave",
					Emoji:    &discordgo.ComponentEmoji{Name: "👋"},
				},

				// visible for everyone; just the admins can use this button
				discordgo.Button{
					Label:    "Admin",
					Style:    discordgo.SecondaryButton,
					CustomID: "admin_panel",
					Emoji:    &discordgo.ComponentEmoji{Name: "👮"},
				},
			},
		},
	}
}

// admin selectors, actions(reset/close) | kick
func AdminComponentsForQueues(qs []*queue.Queue) []discordgo.MessageComponent {
	comps := make([]discordgo.MessageComponent, 0, 2)

	// actions by queue (reset/close), cap of 12 queues (24 options)
	if len(qs) > 0 {
		n := min(len(qs), 12)
		opts := make([]discordgo.SelectMenuOption, 0, n*2)
		for idx := range n {
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

	//Kick select (25 players)
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
