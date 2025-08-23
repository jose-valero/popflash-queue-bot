package config

import (
	"log"
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

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		Token:     os.Getenv("DISCORD_BOT_TOKEN"),
		AppID:     os.Getenv("DISCORD_APP_ID"),
		GuildID:   os.Getenv("DISCORD_GUILD_ID"),
		Prefix:    os.Getenv("DISCORD_PREFIX"),
		ChannelID: os.Getenv("DISCORD_CHANNEL_ID"),
	}

	if cfg.Token == "" || cfg.AppID == "" {
		log.Fatal("Faltan DISCORD_BOT_TOKEN o DISCORD_APP_ID en .env")
	}
	if cfg.GuildID == "" {
		log.Println("⚠️ DISCORD_GUILD_ID vacío: los commands tardarán en propagarse si los registras globales.")
	}
	return cfg
}
