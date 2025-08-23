package discord

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

var (
	qman            = queue.NewManager()
	targetChannelID string
	defaultCapacity = 5
)

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

// -------- slash --------

func handleSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// restringir canal
	if targetChannelID != "" && i.ChannelID != targetChannelID {
		_ = SendEphemeral(s, i, "Use this command in the designated queue channel.")
		return
	}

	queueID := i.ChannelID
	name := i.ApplicationCommandData().Name
	log.Printf("[slash] %s in channel %s", name, i.ChannelID)

	switch name {
	case "startqueue":
		if _, err := qman.CreateQueue(queueID, "Ellos la llevan", defaultCapacity); err != nil {
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}
		q, _ := qman.GetQueue(queueID)
		_ = SendEmbedWithComponents(s, i, renderQueueEmbed(q), componentsWithKick(q))
		return

	case "joinqueue":
		u := userOf(i)
		if u == nil {
			_ = SendEphemeral(s, i, "⚠️ Could not identify the user.")
			return
		}
		if err := qman.JoinQueue(queueID, u.ID, u.Username); err != nil {
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}
		_ = SendResponse(s, i, fmt.Sprintf("🙌 %s joined the queue.", u.Username))

	case "leavequeue":
		u := userOf(i)
		if u == nil {
			_ = SendEphemeral(s, i, "⚠️ Could not identify the user.")
			return
		}
		if err := qman.LeaveQueue(queueID, u.ID); err != nil {
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}
		_ = SendResponse(s, i, fmt.Sprintf("👋 %s left the queue.", u.Username))

	case "queue":
		q, err := qman.GetQueue(queueID)
		if err != nil {
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}
		_ = SendEmbedWithComponents(s, i, renderQueueEmbed(q), nil)
	}
}

// -------- botones --------

func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if targetChannelID != "" && i.ChannelID != targetChannelID {
		_ = SendEphemeral(s, i, "Use the buttons in the designated queue channel.")
		return
	}

	queueID := i.ChannelID
	customID := i.MessageComponentData().CustomID
	u := userOf(i)
	log.Printf("[component] %s by %s", customID, safeName(u))

	// Kick dinámico: botones con CustomID = "kick_<userID>"
	if strings.HasPrefix(customID, "kick_") {
		targetID := strings.TrimPrefix(customID, "kick_")

		if err := qman.LeaveQueue(queueID, targetID); err != nil {
			switch {
			case errors.Is(err, queue.ErrNotIn):
				_ = SendEphemeral(s, i, "⚠️ Ese usuario no está en la cola.")
			case errors.Is(err, queue.ErrNotFound):
				_ = SendEphemeral(s, i, "⚠️ No hay cola activa.")
			default:
				_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			}
			return
		}

		if q, e := qman.GetQueue(queueID); e == nil {
			// Actualiza el embed y re-renderiza todos los botones (controles + kicks)
			_ = UpdateEmbedWithComponents(s, i, renderQueueEmbed(q), componentsWithKick(q))
		} else {
			// Fallback raro: confirma y deja sin tocar componentes
			_ = UpdateMessageWithComponents(s, i, "✅ Usuario removido.", nil)
		}
		return
	}

	switch customID {

	case "queue_join":
		if u == nil {
			_ = SendEphemeral(s, i, "⚠️ Could not identify the user.")
			return
		}
		if err := qman.JoinQueue(queueID, u.ID, u.Username); err != nil {
			switch {
			case errors.Is(err, queue.ErrFull):
				_ = SendEphemeral(s, i, "⚠️ La lista esta llena, te toca esperar sorry.")
			case errors.Is(err, queue.ErrAlreadyIn):
				_ = SendEphemeral(s, i, "⚠️ Ya la estas llevando, calmate.")
			default:
				_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			}
			return
		}
		if q, e := qman.GetQueue(queueID); e == nil {
			_ = UpdateEmbedWithComponents(s, i, renderQueueEmbed(q), componentsWithKick(q))
		}

	case "queue_leave":
		if u == nil {
			_ = SendEphemeral(s, i, "⚠️ Could not identify the user.")
			return
		}
		if err := qman.LeaveQueue(queueID, u.ID); err != nil {
			switch {
			case errors.Is(err, queue.ErrNotIn):
				_ = SendEphemeral(s, i, "⚠️ You are not in the queue.")
			default:
				_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			}
			return
		}
		if q, e := qman.GetQueue(queueID); e == nil {
			_ = UpdateEmbedWithComponents(s, i, renderQueueEmbed(q), componentsWithKick(q))
		}

	case "queue_close":
		qman.DeleteQueue(queueID)
		_ = UpdateMessageWithComponents(s, i, "🛑 Queue closed.", nil)
		return

	case "queue_reset":
		if err := qman.ResetQueue(queueID); err != nil {
			_ = SendEphemeral(s, i, "⚠️ No hay una cola activa que resetear.")
			return
		}

		if q, e := qman.GetQueue(queueID); e == nil {
			_ = UpdateEmbedWithComponents(s, i, renderQueueEmbed(q), componentsWithKick(q))
		} else {
			_ = UpdateEmbedWithComponents(s, i, renderQueueEmbed(nil), componentsRowDisabled(false))
			return
		}
		return

	}
}
