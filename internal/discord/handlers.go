package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

var qManager = queue.NewManager()

// Registrar handlers de comandos
func RegisterHandlers(s *discordgo.Session) {
	s.AddHandler(onInteractionCreate)
}

func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "startqueue":
		handleStartQueue(s, i)
	case "joinqueue":
		handleJoinQueue(s, i)
	case "leavequeue":
		handleLeaveQueue(s, i)
	case "queue":
		handleQueueStatus(s, i)
	}
}

func handleStartQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_, err := qManager.CreateQueue("q1", "Test Queue", 5)
	if err != nil {
		SendResponse(s, i, fmt.Sprintf("⚠️ %s", err.Error()))
		return
	}
	SendResponse(s, i, "✅ Cola creada con éxito (capacidad 5)")
}

func handleJoinQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := i.Member.User
	err := qManager.JoinQueue("q1", user.ID, user.Username)
	if err != nil {
		SendResponse(s, i, fmt.Sprintf("⚠️ %s", err.Error()))
		return
	}
	SendResponse(s, i, fmt.Sprintf("🙌 %s se unió a la cola!", user.Username))
}

func handleLeaveQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := i.Member.User
	err := qManager.LeaveQueue("q1", user.ID)
	if err != nil {
		SendResponse(s, i, fmt.Sprintf("⚠️ %s", err.Error()))
		return
	}
	SendResponse(s, i, fmt.Sprintf("👋 %s salió de la cola.", user.Username))
}

func handleQueueStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
	q, err := qManager.GetQueue("q1")
	if err != nil {
		SendResponse(s, i, fmt.Sprintf("⚠️ %s", err.Error()))
		return
	}

	if len(q.Players) == 0 {
		SendResponse(s, i, "La cola está vacía.")
		return
	}

	msg := fmt.Sprintf("📋 **%s** (%d/%d)\n", q.Name, len(q.Players), q.Capacity)
	for _, p := range q.Players {
		msg += fmt.Sprintf("- %s\n", p.Username)
	}
	SendResponse(s, i, msg)
}
