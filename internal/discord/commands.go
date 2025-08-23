package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var commands = []*discordgo.ApplicationCommand{
	{Name: "startqueue", Description: "create a new queue (test)"},
	{Name: "joinqueue", Description: "join the queue"},
	{Name: "leavequeue", Description: "leave the queue"},
	{Name: "queue", Description: "show queue status"},
}

// Borra comandos en guild y global para evitar duplicados.
// Luego crea SOLO en guild.
func RegisterCommands(s *discordgo.Session, guildID string) error {
	appID := s.State.User.ID

	// 1) Cleanup en ambos scopes: guild y global
	for _, scope := range []string{guildID, ""} {
		if existing, err := s.ApplicationCommands(appID, scope); err == nil {
			for _, cmd := range existing {
				_ = s.ApplicationCommandDelete(appID, scope, cmd.ID)
			}
		}
	}

	// 2) Crear SOLO en el guild (aparecen al instante)
	for _, cmd := range commands {
		if _, err := s.ApplicationCommandCreate(appID, guildID, cmd); err != nil {
			return fmt.Errorf("failed creating %q: %w", cmd.Name, err)
		}
	}
	return nil
}
