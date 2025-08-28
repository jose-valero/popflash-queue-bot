package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Token             string
	AppID             string
	GuildID           string
	Prefix            string
	QueueChannelID    string // dónde renderizamos la UI / botones
	AnnounceChannelID string // de dónde leemos los embeds de PopFlash
	PopflashBase      string
	PopflashToken     string
	FFActiveMatchesUI bool
	PollSeconds       int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	base := firstNonEmpty(os.Getenv("POPFLASH_BASE"), os.Getenv("POPFLASH_API_BASE"))
	tok := firstNonEmpty(os.Getenv("POPFLASH_TOKEN"), os.Getenv("POPFLASH_API_TOKEN"))

	cfg := &Config{
		Token:             os.Getenv("DISCORD_BOT_TOKEN"),
		AppID:             os.Getenv("DISCORD_APP_ID"),
		GuildID:           os.Getenv("DISCORD_GUILD_ID"),
		Prefix:            firstNonEmpty(os.Getenv("DISCORD_PREFIX"), "!"),
		QueueChannelID:    os.Getenv("DISCORD_CHANNEL_ID"),
		AnnounceChannelID: os.Getenv("PF_ANNOUNCE_CHANNEL_ID"),
		PopflashBase:      base,
		PopflashToken:     tok,

		// Feature Flag
		FFActiveMatchesUI: strings.EqualFold(os.Getenv("FF_ACTIVE_MATCHES_UI"), "true"),
		PollSeconds:       parseInt(os.Getenv("PF_POLL_SECONDS"), 60),
	}

	if cfg.Token == "" {
		return nil, errors.New("missing DISCORD_BOT_TOKEN")
	}
	if cfg.AppID == "" {
		return nil, errors.New("missing DISCORD_APP_ID")
	}
	if cfg.GuildID == "" {
		return nil, errors.New("missing DISCORD_GUILD_ID")
	}
	if cfg.QueueChannelID == "" {
		return nil, errors.New("missing DISCORD_QUEUE_CHANNEL_ID (or legacy DISCORD_CHANNEL_ID)")
	}
	if cfg.AnnounceChannelID == "" {
		return nil, errors.New("missing POPFLASH_ANNOUNCE_CHANNEL_ID (or PF_ANNOUNCE_CHANNEL_ID)")
	}

	return cfg, nil
}

func firstNonEmpty(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

// func parseBool(v string) bool {
// 	switch strings.ToLower(strings.TrimSpace(v)) {
// 	case "1", "t", "true", "yes", "y", "on":
// 		return true
// 	default:
// 		return false
// 	}
// }

func parseInt(v string, def int) int {
	if v == "" {
		return def
	}
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return n
	}
	return def
}

func (c *Config) Redacted() string {
	tok := "[set]"
	if c.Token == "" {
		tok = "[empty]"
	}
	return fmt.Sprintf(
		"appID=%s guildID=%s prefix=%q queueChannelID=%s announceChannelID=%s token=%s",
		c.AppID, c.GuildID, c.Prefix, c.QueueChannelID, c.AnnounceChannelID, tok,
	)
}
