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

func RegisterCommands(s *discordgo.Session, guildID string) error {
	appID := s.State.User.ID
	scope := guildID // "" => global, guildID => por servidor

	// limpiar existentes (evita basura)
	if existing, err := s.ApplicationCommands(appID, scope); err == nil {
		for _, cmd := range existing {
			_ = s.ApplicationCommandDelete(appID, scope, cmd.ID)
		}
	}
	// crear
	for _, cmd := range commands {
		if _, err := s.ApplicationCommandCreate(appID, scope, cmd); err != nil {
			return fmt.Errorf("failed creating %q: %w", cmd.Name, err)
		}
	}
	return nil
}
