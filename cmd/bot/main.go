package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	botdiscord "github.com/jose-valero/popflash-queue-bot/internal/discord"
)

func main() {
	_ = godotenv.Load()

	token := os.Getenv("DISCORD_BOT_TOKEN")
	appID := os.Getenv("DISCORD_APP_ID")
	guild := os.Getenv("DISCORD_GUILD_ID")
	channel := os.Getenv("DISCORD_CHANNEL_ID")

	if token == "" || appID == "" {
		log.Fatal("Faltan DISCORD_BOT_TOKEN o DISCORD_APP_ID")
	}

	// capacidad
	capacity := 5
	if v := os.Getenv("QUEUE_CAPACITY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			capacity = n
		}
	}
	botdiscord.SetRuntimeConfig(channel, capacity)

	// sesión
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Error al crear sesión de Discord: %v", err)
	}

	// Intents: Guilds (slash) + GuildMessages (solo para el ping temporal)
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates | discordgo.IntentsMessageContent

	// ping temporal (puedes quitarlo luego)
	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}
		if m.Content == "!ping" {
			_, _ = s.ChannelMessageSend(m.ChannelID, "pong")
		}
	})

	// handler principal
	session.AddHandler(botdiscord.HandleInteraction)
	session.AddHandler(botdiscord.HandleVoiceStateUpdate)
	session.AddHandler(botdiscord.HandleGuildCreate)
	session.AddHandler(botdiscord.HandleMessageCreate)
	session.AddHandler(botdiscord.HandleMessageUpdate)

	// abrir
	if err = session.Open(); err != nil {
		log.Fatalf("Error al abrir conexión: %v", err)
	}
	defer session.Close()

	log.Println("🤖 Bot corriendo. Registrando comandos...")

	// IMPORTANTE: registrar comandos con el BOT_ID (lo hace la función via s.State.User.ID)
	if err := botdiscord.RegisterCommands(session, guild); err != nil {
		log.Fatalf("Error registrando comandos: %v", err)
	}
	log.Println("✅ Comandos listos: /startqueue /joinqueue /leavequeue /queue")

	// esperar señal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}
