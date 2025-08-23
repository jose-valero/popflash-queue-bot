package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	botdiscord "github.com/jose-valero/popflash-queue-bot/internal/discord"
	"github.com/jose-valero/popflash-queue-bot/pkg/config"
)

func main() {
	cfg := config.Load()

	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		log.Fatalf("Error creating session: %v", err)
	}

	if err := s.Open(); err != nil {
		log.Fatalf("Error opening connection: %v", err)
	}
	fmt.Println("🤖 Bot running. registering commands...")

	cleanup, err := botdiscord.Register(s, cfg.AppID, cfg.GuildID)
	if err != nil {
		log.Fatalf("Error registering commands: %v", err)
	}
	defer cleanup()

	fmt.Println("✅ commands listos: /startqueue /joinqueue /leavequeue /queue")

	// Esperar señal para salir
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	_ = s.Close()
}
