package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

const ephemeralFlag = 1 << 6

// --------- helpers de respuesta de texto ---------

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

func SendEphemeralWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, msg string, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    msg,
			Components: comps,
			Flags:      ephemeralFlag,
		},
	})
	if err != nil {
		log.Printf("SendEphemeralWithComponents error: %v", err)
	}
	return err
}

// --------- helpers de EMBED ---------

func SendEmbedWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, emb *discordgo.MessageEmbed, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{emb},
			Components: comps,
		},
	})
	if err != nil {
		log.Printf("SendEmbedWithComponents error: %v", err)
	}
	return err
}

func UpdateEmbedWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, emb *discordgo.MessageEmbed, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{emb},
			Components: comps,
		},
	})
	if err != nil {
		log.Printf("UpdateEmbedWithComponents error: %v", err)
	}
	return err
}

func SendEphemeralEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, emb *discordgo.MessageEmbed) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{emb},
			Flags:  ephemeralFlag,
		},
	})
	if err != nil {
		log.Printf("SendEphemeralEmbed error: %v", err)
	}
	return err
}

// --------- update simple de texto (por completitud) ---------

func UpdateMessageWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, content string, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: comps,
		},
	})
	if err != nil {
		log.Printf("UpdateMessageWithComponents error: %v", err)
	}
	return err
}

// --------- util de usuario ---------

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
