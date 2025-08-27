package app

import (
	"log"
	"sync"

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

var wiringOnce sync.Once

func (b *Bot) RegisterHandlers() {
	wiringOnce.Do(func() {
		SetRuntimeConfig(b.Cfg.QueueChannelID, 5)

		b.Sess.AddHandler(disc.TrackVoiceState)

		disc.SetAnnounceChannel(b.Cfg.AnnounceChannelID)
		b.Sess.AddHandler(disc.HandleMessageCreate)
		b.Sess.AddHandler(disc.HandleMessageUpdate)

		b.Sess.AddHandler(HandleInteraction) // slash + components

		b.cancelBus = b.StartEventSubscribers()

		_ = RegisterCommands(b.Sess, b.Cfg.AppID, b.Cfg.GuildID)
		log.Printf("[wiring] handlers registered (once)")
	})
}

func (b *Bot) Stop() {
	if b.cancelBus != nil {
		b.cancelBus()
	}
}
