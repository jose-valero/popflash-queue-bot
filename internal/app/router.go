// internal/app/router.go
package app

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"

	d "github.com/jose-valero/popflash-queue-bot/internal/adapters/discord"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
	"github.com/jose-valero/popflash-queue-bot/internal/ui"
)

var (
	qman            = queue.NewManager()
	targetChannelID string
	defaultCapacity = 5
)

// channel open/close flag for joins
var queueOpen sync.Map // channelID -> bool

func SetQueueOpen(channelID string, open bool) {
	if channelID == "" {
		return
	}
	queueOpen.Store(channelID, open)
}

func IsQueueOpen(channelID string) bool {
	if v, ok := queueOpen.Load(channelID); ok {
		if b, ok2 := v.(bool); ok2 {
			return b
		}
	}
	return false
}

func SetRuntimeConfig(channelID string, capacity int) {
	targetChannelID = channelID
	if capacity > 0 {
		defaultCapacity = capacity
	}
}

func HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleSlash(s, i)
	case discordgo.InteractionMessageComponent:
		handleComponent(s, i)
	}
}

// ------------------- Slash -------------------

func handleSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if targetChannelID != "" && i.ChannelID != targetChannelID {
		_ = d.SendEphemeral(s, i, "Use this command in the designated queue channel.")
		return
	}

	queueID := i.ChannelID
	name := i.ApplicationCommandData().Name
	log.Printf("[slash] %s in channel %s", name, i.ChannelID)

	switch name {

	case "startqueue":
		if !d.IsPrivileged(i) {
			_ = d.SendEphemeral(s, i, "Solo admins pueden abrir la cola.")
			return
		}
		if _, err := qman.EnsureFirstQueue(queueID, "Queue #1", defaultCapacity); err != nil {
			_ = d.SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}
		// abrir ANTES de renderizar para que salga habilitado el boton
		SetQueueOpen(queueID, true)

		// 1) Responder EFÍMERO para cumplir el ACK en <3s (no crea mensaje público)
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "✅ Queue ready.",
			},
		}); err != nil {
			log.Printf("respond error: %v", err)
			return
		}

		// 2) Publicar/editar el mensaje PÚBLICO con nuestra UI y RECORDAR SU ID
		qs, _ := qman.Queues(queueID)
		if err := d.PublishOrEditQueueMessage(
			s, queueID,
			ui.RenderQueuesEmbed(qs),
			ui.ComponentsForQueues(qs, IsQueueOpen(queueID)),
		); err != nil {
			log.Printf("PublishOrEditQueueMessage error: %v", err)
		}
		return

	case "joinqueue":
		u := d.UserOf(i)

		if u == nil {
			_ = d.SendEphemeral(s, i, "⚠️ Could not identify you.")
			return
		}
		if !IsQueueOpen(queueID) {
			_ = d.SendEphemeral(s, i, "🔒 Queue is closed. Wait for the next **match started**.")
			return
		}
		if d.VoiceRequireToJoin() && !d.IsUserInAllowedVoice(s, i.GuildID, u.ID) {
			_ = d.SendEphemeral(s, i, "🔇 You must be in an allowed voice channel to join.")
			return
		}
		if _, err := qman.JoinAny(queueID, u.ID, u.Username, defaultCapacity); err != nil {
			if errors.Is(err, queue.ErrAlreadyIn) {
				_ = d.SendEphemeral(s, i, "You're already in a queue.")
				return
			}
			_ = d.SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}
		_ = d.SendEphemeral(s, i, "🙌 Done! Added you to the first queue with space.")
		updateUIAfterChange(s, i, queueID)
		return

	case "leavequeue":
		u := d.UserOf(i)
		if u == nil {
			_ = d.SendEphemeral(s, i, "⚠️ Could not identify you.")
			return
		}
		if _, err := qman.LeaveAny(queueID, u.ID); err != nil {
			switch {
			case errors.Is(err, queue.ErrNotIn):
				_ = d.SendEphemeral(s, i, "⚠️ You're not in any queue.")
			default:
				_ = d.SendEphemeral(s, i, "⚠️ "+err.Error())
			}
			return
		}
		_ = d.SendEphemeral(s, i, "👋 Left your queue and re-balanced lists.")
		updateUIAfterChange(s, i, queueID)
		return

	case "queue":
		if qs, err := qman.Queues(queueID); err == nil {
			if d.IsPrivileged(i) {
				// Admin: embed + selects solo para él (efímero)
				_ = d.SendEphemeralComplex(s, i, ui.RenderQueuesEmbed(qs), ui.AdminComponentsForQueues(qs))
			} else {
				// No admin: solo embed efímero (sin selects)
				_ = d.SendEphemeralEmbed(s, i, ui.RenderQueuesEmbed(qs))
			}
		} else {
			_ = d.SendEphemeral(s, i, "⚠️ No active queues.")
		}
		return
	}
}

// ------------------- Components -------------------

func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if targetChannelID != "" && i.ChannelID != targetChannelID {
		_ = d.SendEphemeral(s, i, "Use buttons in the designated queue channel.")
		return
	}

	queueID := i.ChannelID
	customID := i.MessageComponentData().CustomID
	u := d.UserOf(i)
	log.Printf("[component] %s by %s", customID, d.SafeName(u))

	// Select: actions per queue ("reset:N" / "close:N")

	if customID == "queue_action" {
		if !d.RequirePrivileged(s, i) {
			return
		}
		vals := i.MessageComponentData().Values
		if len(vals) == 0 {
			_ = d.SendEphemeral(s, i, "⚠️ Invalid selection.")
			return
		}
		parts := strings.SplitN(vals[0], ":", 2)
		if len(parts) != 2 {
			_ = d.SendEphemeral(s, i, "⚠️ Invalid selection.")
			return
		}
		idx, _ := strconv.Atoi(parts[1])

		var err error
		switch parts[0] {
		case "reset":
			err = qman.ResetAt(queueID, idx)
		case "close":
			err = qman.DeleteAt(queueID, idx)
		default:
			_ = d.SendEphemeral(s, i, "⚠️ Unknown action.")
			return
		}
		if err != nil {
			_ = d.SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}

		_ = d.SendEphemeral(s, i, "✅ Done.")
		updateUIAfterChange(s, i, queueID)
		return
	}

	// Select: kick ("uid:<userID>")
	if customID == "queue_kick" {
		if !d.RequirePrivileged(s, i) {
			return
		}
		vals := i.MessageComponentData().Values
		if len(vals) == 0 {
			_ = d.SendEphemeral(s, i, "⚠️ Invalid selection.")
			return
		}
		uid := strings.TrimPrefix(vals[0], "uid:")

		if _, err := qman.LeaveAny(queueID, uid); err != nil {
			switch {
			case errors.Is(err, queue.ErrNotIn):
				_ = d.SendEphemeral(s, i, "⚠️ That user is not in any queue.")
			case errors.Is(err, queue.ErrNotFound):
				_ = d.SendEphemeral(s, i, "⚠️ No active queues.")
			default:
				_ = d.SendEphemeral(s, i, "⚠️ "+err.Error())
			}
			return
		}

		_ = d.SendEphemeral(s, i, "✅ Player kicked.")
		updateUIAfterChange(s, i, queueID)
		return
	}

	switch customID {

	case "queue_join":
		if u == nil {
			_ = d.SendEphemeral(s, i, "⚠️ Could not identify you.")
			return
		}
		if !IsQueueOpen(queueID) {
			_ = d.SendEphemeral(s, i, "🔒 Queue is closed. Wait for the next **match started**.")
			return
		}
		if d.VoiceRequireToJoin() && !d.IsUserInAllowedVoice(s, i.GuildID, u.ID) {
			_ = d.SendEphemeral(s, i, "🔇 You must be in an allowed voice channel to join.")
			return
		}
		if _, err := qman.JoinAny(queueID, u.ID, u.Username, defaultCapacity); err != nil {
			if errors.Is(err, queue.ErrAlreadyIn) {
				_ = d.SendEphemeral(s, i, "You're already in a queue.")
				return
			}
			_ = d.SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}
		_ = d.SendEphemeral(s, i, "🙌 Joined!")
		updateUIAfterChange(s, i, queueID)
		return

	case "queue_leave":
		if u == nil {
			_ = d.SendEphemeral(s, i, "⚠️ Could not identify you.")
			return
		}
		if _, err := qman.LeaveAny(queueID, u.ID); err != nil {
			switch {
			case errors.Is(err, queue.ErrNotIn):
				_ = d.SendEphemeral(s, i, "⚠️ You're not in any queue.")
			default:
				_ = d.SendEphemeral(s, i, "⚠️ "+err.Error())
			}
			return
		}
		_ = d.SendEphemeral(s, i, "👋 Left.")
		updateUIAfterChange(s, i, queueID)
		return

	case "admin_panel":
		if !d.RequirePrivileged(s, i) {
			return
		}
		if qs, err := qman.Queues(queueID); err == nil && len(qs) > 0 {
			_ = d.SendEphemeralComplex(
				s, i,
				ui.RenderQueuesEmbed(qs),
				ui.AdminComponentsForQueues(qs),
			)
		} else {
			_ = d.SendEphemeral(s, i, "⚠️ No active queues.")
		}
		return
	}

}

// updateUIAfterChange refreshes the public embed+components OUTSIDE the interaction.
// If the manager reports no queues (ErrNotFound), we ensure Queue #1 exists and
// render it EMPTY (0/N) instead of showing the “No queues” embed.
func updateUIAfterChange(s *discordgo.Session, _ *discordgo.InteractionCreate, channelID string) {
	var (
		qs  []*queue.Queue
		err error
	)

	qs, err = qman.Queues(channelID)
	if errors.Is(err, queue.ErrNotFound) {
		// Make sure we always show “Queue #1 (0/N)” rather than “No queues”.
		if q, e2 := qman.EnsureFirstQueue(channelID, "Queue #1", defaultCapacity); e2 == nil && q != nil {
			qs = []*queue.Queue{q}
			err = nil
		}
	}

	var emb *discordgo.MessageEmbed
	var comps []discordgo.MessageComponent

	if err == nil && len(qs) > 0 {
		emb = ui.RenderQueuesEmbed(qs)
		comps = ui.ComponentsForQueues(qs, IsQueueOpen(channelID))
	} else {
		// Fallback (shouldn’t normally happen after EnsureFirstQueue)
		emb = ui.RenderQueuesEmbed(nil)
		comps = ui.ComponentsForQueues(nil, IsQueueOpen(channelID))
	}

	if err2 := d.EditQueueMessage(s, channelID, emb, comps); err2 != nil {
		log.Printf("updateUIAfterChange: edit failed: %v", err2)
	}
}
