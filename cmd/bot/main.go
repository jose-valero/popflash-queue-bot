// Command bot start the discord bot process
//
// this binary:
//  1. Load config from enviroment variables (.env during dev)
//  2. creates a discord session
//  3. registers the app handlers
//  4. open connection to gatewaya connection and waits a signal from OS to exit
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"github.com/jose-valero/popflash-queue-bot/internal/app"
	"github.com/jose-valero/popflash-queue-bot/pkg/config"
)

func main() {
	// load .env from local development.
	_ = godotenv.Load()

	// read and validate the minimal config to work
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// create a session of discord
	//  the prefix "Bot " is required form bot tokens
	sess, err := discordgo.New("Bot " + cfg.Token)

	if err != nil {
		log.Fatalf("discord session error: %v", err)
	}

	// ⬇️ Intents necesarios
	sess.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages | // para MessageCreate/Update en canales
		discordgo.IntentsMessageContent | // para leer "match started/finished" en el texto
		discordgo.IntentsGuildVoiceStates // si usas la política de voz

	// instance the app Bot and register all handlers
	// this layer keeps wiring serparete from domain
	b := app.NewBot(sess, cfg)
	b.RegisterHandlers()

	// open websocket gateway
	if err := sess.Open(); err != nil {
		log.Fatalf("open gateway error: %v", err)
	}

	defer sess.Close() //--> close de connection to leave

	log.Printf("🤖 bot ready - %s", cfg.Redacted())

	// block the process till get SIGINT/SIGTEM
	// this allow a clean shutdown (Ctrl+c, kill, etc)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}
