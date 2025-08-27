// internal/app/commands.go
package app

import "github.com/bwmarrin/discordgo"

var adminPerms int64 = discordgo.PermissionAdministrator

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "startqueue",
		Description: "Create/open the queue UI in this channel",
		Type:        discordgo.ChatApplicationCommand,
	},
	{
		Name:        "joinqueue",
		Description: "Join the first queue with space",
		Type:        discordgo.ChatApplicationCommand,
	},
	{
		Name:        "leavequeue",
		Description: "Leave whatever queue you're in",
		Type:        discordgo.ChatApplicationCommand,
	},
	{
		Name:        "queue",
		Description: "Show queue status",
		Type:        discordgo.ChatApplicationCommand,
	},
	{
		Name:                     "seedqueue",
		Description:              "Agrega N jugadores mock a las colas (dev only)",
		Type:                     discordgo.ChatApplicationCommand,
		DefaultMemberPermissions: &adminPerms,

		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "n",
				Description: "Cantidad de jugadores mock a agregar",
				Required:    false, // default 12
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prefix",
				Description: "Prefijo de los mocks (default: mock)",
				Required:    false,
			},
		},
	},
	{
		Name:                     "clearmocks",
		Description:              "Quita todos los jugadores mock de las colas (dev only)",
		Type:                     discordgo.ChatApplicationCommand,
		DefaultMemberPermissions: &adminPerms,
	},
}

// RegisterCommands creates (or updates) guild-level commands.
func RegisterCommands(s *discordgo.Session, appID, guildID string) error {
	_, err := s.ApplicationCommandBulkOverwrite(appID, guildID, commands)
	return err
}
