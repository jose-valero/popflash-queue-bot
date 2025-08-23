package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

const ephemeralFlag = 1 << 6

func SendResponse(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg},
	})
	if err != nil {
		log.Printf("SendResponse error: %v", err)
	}
	return err
}

// Respuesta con componentes (botones)
func SendResponseWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, msg string, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    msg,
			Components: comps,
		},
	})
	if err != nil {
		log.Printf("SendResponseWithComponents error: %v", err)
	}
	return err
}

// Respuesta efímera (solo quien interactúa la ve)
func SendEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   ephemeralFlag,
		},
	})
	if err != nil {
		log.Printf("SendEphemeral error: %v", err)
	}
	return err
}

// Update del mensaje original (para clicks en botones)
func UpdateMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: componentsRow(),
		},
	})
	if err != nil {
		log.Printf("UpdateMessage error: %v", err)
	}
	return err
}

func userOf(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	return i.User
}

func safeName(u *discordgo.User) string {
	if u == nil {
		return "unknown"
	}
	return u.Username
}
