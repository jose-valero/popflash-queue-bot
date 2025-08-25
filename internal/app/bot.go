// internal/app/bot.go
// Package app contains the application “shell”:
// high-level types, wiring (handler registration) and orchestration.
// The goal is to keep domain logic OUT of this package; this is glue.

package app

import (
	"github.com/bwmarrin/discordgo"

	"github.com/jose-valero/popflash-queue-bot/pkg/config"
)

// Bot represents a running bot instance.
// It holds the Discord session and the already-parsed configuration.
type Bot struct {
	// Sess is the connected Discord session (Gateway/REST client).
	Sess *discordgo.Session
	// Cfg is the application configuration (IDs, token, etc.).
	Cfg *config.Config
}

// NewBot builds a Bot from a Discord session and configuration.
// It does NOT register handlers or open connections; see RegisterHandlers/main.
func NewBot(s *discordgo.Session, cfg *config.Config) *Bot {
	return &Bot{Sess: s, Cfg: cfg}
}

// RegisterHandlers wires the app-level handlers to the Discord session.
// Right now we only register the interaction router (slash/buttons/selects).
// PopFlash message listeners and voice handlers will be added in a later step.
func (b *Bot) RegisterHandlers() {
	// Runtime config for the queue channel + per-queue capacity.
	SetRuntimeConfig(b.Cfg.ChannelID, 5)

	// Interactions (slash/buttons/selects).
	b.Sess.AddHandler(HandleInteraction)

	// TODO: when we move PopFlash/Voice code to adapters, add:
	// b.Sess.AddHandler(HandleMessageCreate)
	// b.Sess.AddHandler(HandleMessageUpdate)
	// b.Sess.AddHandler(HandleVoiceStateUpdate)

	// Optionally (recommended): ensure slash commands are registered.
	_ = RegisterCommands(b.Sess, b.Cfg.AppID, b.Cfg.GuildID)
}
