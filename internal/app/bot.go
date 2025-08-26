package app

import (
	"github.com/bwmarrin/discordgo"

	disc "github.com/jose-valero/popflash-queue-bot/internal/adapters/discord"
	"github.com/jose-valero/popflash-queue-bot/pkg/config"
)

type Bot struct {
	Sess      *discordgo.Session
	Cfg       *config.Config
	cancelBus func()
}

func NewBot(s *discordgo.Session, cfg *config.Config) *Bot {
	return &Bot{Sess: s, Cfg: cfg}
}

func (b *Bot) RegisterHandlers() {
	// 1) Config de la UI de colas
	SetRuntimeConfig(b.Cfg.QueueChannelID, 5)

	// 2) Voice tracking (para VOICE_REQUIRE_TO_JOIN si lo usas)
	b.Sess.AddHandler(disc.TrackVoiceState)

	// 3) PRODUCTOR: listeners del canal de anuncios (tu announcer.go)
	disc.SetAnnounceChannel(b.Cfg.AnnounceChannelID)
	b.Sess.AddHandler(disc.HandleMessageCreate)
	b.Sess.AddHandler(disc.HandleMessageUpdate)

	// 4) Router de interacciones (slash/buttons/selects)
	b.Sess.AddHandler(HandleInteraction)

	// 5) SUSCRIPTORES del bus: abren/cierran la cola
	b.cancelBus = b.StartEventSubscribers()

	// 6)registrar/actualizar slash commands
	_ = RegisterCommands(b.Sess, b.Cfg.AppID, b.Cfg.GuildID)
}

// Llamable desde main si quieres parar subs limpio (opcional)
func (b *Bot) Stop() {
	if b.cancelBus != nil {
		b.cancelBus()
	}
}
