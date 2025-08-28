package app

import (
	"log"
	"sync"

	"github.com/bwmarrin/discordgo"
	disc "github.com/jose-valero/popflash-queue-bot/internal/adapters/discord"
	"github.com/jose-valero/popflash-queue-bot/internal/adapters/popflash"
	"github.com/jose-valero/popflash-queue-bot/pkg/config"
)

type Bot struct {
	Sess      *discordgo.Session
	Cfg       *config.Config
	PF        *popflash.Client
	cancelBus func()
}

func NewBot(s *discordgo.Session, cfg *config.Config) *Bot {
	pf := popflash.New(cfg.PopflashBase, cfg.PopflashToken)
	if cfg.PopflashToken != "" {
		base := cfg.PopflashBase
		if base == "" {
			base = "https://api.popflash.site" // o el default que use tu client
		}
		pf = popflash.New(base, cfg.PopflashToken)
	}
	return &Bot{Sess: s, Cfg: cfg, PF: pf}
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
		if b.Cfg.PollSeconds > 0 {
			b.StartScorePoller()
		}
		_ = RegisterCommands(b.Sess, b.Cfg.AppID, b.Cfg.GuildID)
		log.Printf("[wiring] handlers registered (once)")
	})
}

func (b *Bot) Stop() {
	if b.cancelBus != nil {
		b.cancelBus()
	}
}
