package discord

import (
	"errors"
	"fmt"
	"log"

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
		if _, err := qman.CreateQueue(queueID, "Match Queue", defaultCapacity); err != nil {
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}
		q, _ := qman.GetQueue(queueID)
		fmt.Println("🧪 q =>", q)
		_ = SendResponseWithComponents(s, i, renderQueue(q), componentsRow())
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
		_ = SendResponse(s, i, renderQueue(q))
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

	switch customID {
	case "queue_join":
		if u == nil {
			_ = SendEphemeral(s, i, "⚠️ Could not identify the user.")
			return
		}
		if err := qman.JoinQueue(queueID, u.ID, u.Username); err != nil {
			switch {
			case errors.Is(err, queue.ErrFull):
				if q, e := qman.GetQueue(queueID); e == nil {
					_ = UpdateMessage(s, i, renderQueue(q)+"\n\n⚠️ Queue is full.")
				} else {
					_ = SendEphemeral(s, i, "⚠️ "+err.Error())
				}
			case errors.Is(err, queue.ErrAlreadyIn):
				_ = SendEphemeral(s, i, "⚠️ You are already in the queue.")
			default:
				_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			}
			return
		}
		if q, e := qman.GetQueue(queueID); e == nil {
			_ = UpdateMessage(s, i, renderQueue(q))
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
			_ = UpdateMessage(s, i, renderQueue(q))
		}

	case "queue_close":
		qman.DeleteQueue(queueID)
		_ = UpdateMessage(s, i, "🛑 Queue closed.")
		return
	}
}
