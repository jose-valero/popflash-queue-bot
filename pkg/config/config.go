package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Token     string
	AppID     string
	GuildID   string
	Prefix    string
	ChannelID string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Token:     os.Getenv("DISCORD_BOT_TOKEN"),
		AppID:     os.Getenv("DISCORD_APP_ID"),
		GuildID:   os.Getenv("DISCORD_GUILD_ID"),
		Prefix:    firstNonEmpty(os.Getenv("DISCORD_PREFIX"), "!"),
		ChannelID: os.Getenv("DISCORD_CHANNEL_ID"),
	}

	//---- minimal validations of envs ----
	if cfg.Token == "" || cfg.AppID == "" {
		return nil, errors.New("missing DISCORD_BOT_TOKEN")
	}

	if cfg.AppID == "" {
		return nil, errors.New("missing DISCORD_APP_ID")
	}

	if cfg.GuildID == "" {
		return nil, errors.New("missing DISCORD_GUILD_ID")
	}

	return cfg, nil
}

// simple helper for non empty value, por ahora para el prefix
func firstNonEmpty(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func (c *Config) Redacted() string {
	tok := "[set]"

	if c.Token == "" {
		tok = "[empyy]"
	}

	return fmt.Sprintf("appID=%s guildID=%s prefix=%q channelID=%s token=%s",
		c.AppID, c.GuildID, c.Prefix, c.ChannelID, tok)
}
