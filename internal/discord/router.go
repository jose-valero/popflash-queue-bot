package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// Register registra los slash commands en un GUILD (si guildID != "")
// y devuelve una función de cleanup para borrarlos al salir.
func Register(s *discordgo.Session, appID, guildID string) (func(), error) {
	created := make([]*discordgo.ApplicationCommand, 0, len(Commands))
	scope := guildID // "" => global; guildID => solo en ese server

	for _, cmd := range Commands {
		ac, err := s.ApplicationCommandCreate(appID, scope, cmd)
		if err != nil {
			return nil, fmt.Errorf("no se pudo registrar %s: %w", cmd.Name, err)
		}
		created = append(created, ac)
	}

	// Handler central (prototipo simulado)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		switch i.ApplicationCommandData().Name {
		case "startqueue":
			SendResponse(s, i, "✅ Cola creada (simulado)")
		case "joinqueue":
			SendResponse(s, i, "🙌 Te uniste a la cola (simulado)")
		case "leavequeue":
			SendResponse(s, i, "👋 Saliste de la cola (simulado)")
		case "queue":
			SendResponse(s, i, "📋 Estado de la cola: (simulado)")
		}
	})

	cleanup := func() {
		for _, cmd := range created {
			if err := s.ApplicationCommandDelete(appID, scope, cmd.ID); err != nil {
				log.Printf("No se pudo borrar command %s: %v", cmd.Name, err)
			}
		}
	}
	return cleanup, nil
}
