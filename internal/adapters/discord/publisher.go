package discord

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

var queueMsgIDs sync.Map // channelID -> messageID

// SetQueueMessageID remembers the last published queue message for a channel.
func SetQueueMessageID(channelID, messageID string) {
	if channelID != "" && messageID != "" {
		queueMsgIDs.Store(channelID, messageID)
	}
}

func getQueueMessageID(channelID string) (string, bool) {
	v, ok := queueMsgIDs.Load(channelID)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// PublishOrEditQueueMessage creates the message the first time, and edits it later.
func PublishOrEditQueueMessage(s *discordgo.Session, channelID string, emb *discordgo.MessageEmbed, comps []discordgo.MessageComponent) error {
	if _, ok := getQueueMessageID(channelID); ok {
		return EditQueueMessage(s, channelID, emb, comps)
	}
	msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{emb},
		Components: comps,
	})
	if err != nil {
		return err
	}
	if msg != nil {
		SetQueueMessageID(channelID, msg.ID)
	}
	return nil
}

// EditQueueMessage updates the remembered message for the channel.
func EditQueueMessage(s *discordgo.Session, channelID string, emb *discordgo.MessageEmbed, comps []discordgo.MessageComponent) error {
	msgID, ok := getQueueMessageID(channelID)
	if !ok {
		return nil
	}
	embeds := []*discordgo.MessageEmbed{emb}
	compsCopy := comps
	_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: channelID, ID: msgID, Embeds: &embeds, Components: &compsCopy,
	})
	if err != nil {
		// Fallback si el mensaje ya no existe (Discord code 10008)
		if re, ok := err.(*discordgo.RESTError); ok && re.Message != nil && re.Message.Code == 10008 {
			queueMsgIDs.Delete(channelID)
			return PublishOrEditQueueMessage(s, channelID, emb, comps)
		}
	}
	return err
}
