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

func mustSingleInstanceLock() func() {
	f, err := os.OpenFile("/tmp/popflashbot.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Fatalf("lock open: %v", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		log.Fatalf("another instance is already running (lock busy)")
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
}

func main() {
	unlock := mustSingleInstanceLock()
	defer unlock()

	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	sess, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		log.Fatalf("discord session error: %v", err)
	}

	sess.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuildVoiceStates

	b := app.NewBot(sess, cfg)
	b.RegisterHandlers()

	if err := sess.Open(); err != nil {
		log.Fatalf("open gateway error: %v", err)
	}
	defer sess.Close()

	log.Printf("🤖 bot ready - %s", cfg.Redacted())

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}
