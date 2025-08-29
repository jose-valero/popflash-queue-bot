package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

const ephemeralFlag = 1 << 6

// SendResponse posts a normal (public) message as the interaction response.
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

// SendEphemeral posts an ephemeral message only visible to the user who interacted.
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

// Responde efímero con SOLO componentes (sin embed).
// Usa un contenido "invisible" para satisfacer la API.
func SendEphemeralComponents(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	comps []discordgo.MessageComponent,
) error {
	// Zero-width space para que el mensaje no muestre texto
	const zwsp = "\u200B"
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral,
			Content:    zwsp,
			Components: comps,
		},
	})
}

// SendEphemeralEmbed responds with an ephemeral embed.
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

func SendEphemeralComplex(s *discordgo.Session, i *discordgo.InteractionCreate, emb *discordgo.MessageEmbed, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{emb},
			Components: comps,
			Flags:      1 << 6, // ephemeral
		},
	})
	if err != nil {
		log.Printf("SendEphemeralComplex error: %v", err)
	}
	return err
}

// UpdateEmbedWithComponents updates the original message of the interaction.
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

// UpdateMessageWithComponents updates the original message with plain text.
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

// UserOf extracts the effective user from an interaction (guild or DM).
func UserOf(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	return i.User
}

// SafeName returns a defensively safe username string.
func SafeName(u *discordgo.User) string {
	if u == nil {
		return "unknown"
	}
	return u.Username
}
