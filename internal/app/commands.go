// internal/app/commands.go
package app

import "github.com/bwmarrin/discordgo"

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
}

// RegisterCommands creates (or updates) guild-level commands.
func RegisterCommands(s *discordgo.Session, appID, guildID string) error {
	_, err := s.ApplicationCommandBulkOverwrite(appID, guildID, commands)
	return err
}
