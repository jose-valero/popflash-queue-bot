package discord

import "github.com/bwmarrin/discordgo"

var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "startqueue",
		Description: "create a new queue (test)",
	},
	{
		Name:        "joinqueue",
		Description: "join the queue",
	},
	{
		Name:        "leavequeue",
		Description: "leave the queue",
	},
	{
		Name:        "queue",
		Description: "show queue status",
	},
}
