package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Token   string
	AppID   string
	GuildID string
	Prefix  string

	// Canales explícitos
	QueueChannelID    string // dónde renderizamos la UI / botones
	AnnounceChannelID string // de dónde leemos los embeds de PopFlash
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	// Back-compat: DISCORD_CHANNEL_ID sigue funcionando para la cola
	// queueID := firstNonEmpty(os.Getenv("DISCORD_QUEUE_CHANNEL_ID"), os.Getenv("DISCORD_CHANNEL_ID"))

	// Preferimos PF_ANNOUNCE_CHANNEL_ID (tu env actual)
	// announceID := firstNonEmpty(os.Getenv("POPFLASH_ANNOUNCE_CHANNEL_ID"), os.Getenv("PF_ANNOUNCE_CHANNEL_ID"))

	cfg := &Config{
		Token:             os.Getenv("DISCORD_BOT_TOKEN"),
		AppID:             os.Getenv("DISCORD_APP_ID"),
		GuildID:           os.Getenv("DISCORD_GUILD_ID"),
		Prefix:            firstNonEmpty(os.Getenv("DISCORD_PREFIX"), "!"),
		QueueChannelID:    os.Getenv("DISCORD_CHANNEL_ID"),
		AnnounceChannelID: os.Getenv("PF_ANNOUNCE_CHANNEL_ID"),
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
