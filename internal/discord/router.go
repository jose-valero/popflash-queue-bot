package discord

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

var (
	qman            = queue.NewManager()
	targetChannelID string
	defaultCapacity = 5
)

// Config en runtime (canal destino y capacidad por cola)
func SetRuntimeConfig(channelID string, capacity int) {
	targetChannelID = channelID
	if capacity > 0 {
		defaultCapacity = capacity
	}
}

// ------------------- la posta ----------------
func HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleSlash(s, i)
	case discordgo.InteractionMessageComponent:
		handleComponent(s, i)
	}
}

// ------------------- SLASH -------------------

func handleSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// restringir canal (opcional)
	if targetChannelID != "" && i.ChannelID != targetChannelID {
		_ = SendEphemeral(s, i, "Usa este comando en el canal designado de la cola.")
		return
	}

	queueID := i.ChannelID
	name := i.ApplicationCommandData().Name
	log.Printf("[slash] %s in channel %s", name, i.ChannelID)

	switch name {

	case "startqueue":
		if _, err := qman.EnsureFirstQueue(queueID, "Queue #1", defaultCapacity); err != nil {
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}
		seedDemoPlayers(11, queueID)
		if qs, err := qman.Queues(queueID); err == nil {
			_ = SendEmbedWithComponents(s, i, renderQueuesEmbed(qs), componentsForQueues(qs))
		}
		return

	case "joinqueue":
		u := userOf(i)
		if u == nil {
			_ = SendEphemeral(s, i, "⚠️ No pudimos identificarte.")
			return
		}
		if _, err := qman.JoinAny(queueID, u.ID, u.Username, defaultCapacity); err != nil && !errors.Is(err, queue.ErrAlreadyIn) {
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}
		_ = SendEphemeral(s, i, "🙌 ¡Listo! Te agregamos a la primera cola con espacio.")
		return

	case "leavequeue":
		u := userOf(i)
		if u == nil {
			_ = SendEphemeral(s, i, "⚠️ No pudimos identificarte.")
			return
		}
		if _, err := qman.LeaveAny(queueID, u.ID); err != nil {
			switch {
			case errors.Is(err, queue.ErrNotIn):
				_ = SendEphemeral(s, i, "⚠️ No estás en ninguna cola.")
			default:
				_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			}
			return
		}
		_ = SendEphemeral(s, i, "👋 Saliste de tu cola y re-balanceamos listas.")
		return

	case "queue":
		if qs, err := qman.Queues(queueID); err == nil {
			_ = SendEphemeralEmbed(s, i, renderQueuesEmbed(qs))
		} else {
			_ = SendEphemeral(s, i, "⚠️ No hay colas activas.")
		}
		return
	}
}

// ------------------- COMPONENTES (botones / selects) -------------------

func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if targetChannelID != "" && i.ChannelID != targetChannelID {
		_ = SendEphemeral(s, i, "Usa los botones en el canal designado de la cola.")
		return
	}

	queueID := i.ChannelID
	customID := i.MessageComponentData().CustomID
	u := userOf(i)
	log.Printf("[component] %s by %s", customID, safeName(u))

	// Select de acciones por cola: "reset:N" / "close:N"
	if customID == "queue_action" {
		// 🔒 permisos
		if !requirePrivileged(s, i) {
			return
		}

		vals := i.MessageComponentData().Values
		if len(vals) == 0 {
			_ = SendEphemeral(s, i, "⚠️ Selección inválida.")
			return
		}
		parts := strings.SplitN(vals[0], ":", 2)
		if len(parts) != 2 {
			_ = SendEphemeral(s, i, "⚠️ Selección inválida.")
			return
		}
		idx, _ := strconv.Atoi(parts[1])

		var err error
		switch parts[0] {
		case "reset":
			err = qman.ResetAt(queueID, idx)
		case "close":
			err = qman.DeleteAt(queueID, idx)
		default:
			_ = SendEphemeral(s, i, "⚠️ Acción desconocida.")
			return
		}
		if err != nil {
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}

		if qs, e := qman.Queues(queueID); e == nil {
			_ = UpdateEmbedWithComponents(s, i, renderQueuesEmbed(qs), componentsForQueues(qs))
		} else {
			// no quedan colas => deja UI mínima para poder volver a "Join"
			_ = UpdateEmbedWithComponents(s, i, renderQueuesEmbed(nil), componentsForQueues(nil))
		}
		return
	}

	// Select "queue_kick" — kickea al jugador elegido (requiere permisos)
	if customID == "queue_kick" {
		// 🔒 permisos: usa roles de ADMIN_ROLE_IDS o Administrator
		if !requirePrivileged(s, i) {
			return
		}

		vals := i.MessageComponentData().Values
		if len(vals) == 0 {
			_ = SendEphemeral(s, i, "⚠️ Selección inválida.")
			return
		}
		uid := vals[0]
		uid = strings.TrimPrefix(uid, "uid:")

		// (opcional) busca el nombre para confirmación
		// var victim string
		if qs, err := qman.Queues(queueID); err == nil {
		outer:
			for _, q := range qs {
				for _, p := range q.Players {
					if p.ID == uid {
						// victim = p.Username
						break outer
					}
				}
			}
		}

		if _, err := qman.LeaveAny(queueID, uid); err != nil {
			switch {
			case errors.Is(err, queue.ErrNotIn):
				_ = SendEphemeral(s, i, "⚠️ Ese usuario ya no está en ninguna cola.")
			case errors.Is(err, queue.ErrNotFound):
				_ = SendEphemeral(s, i, "⚠️ No hay colas activas.")
			default:
				_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			}
			return
		}

		// Re-render UI tras el kick (con rebalanceo automático)
		if qs, err := qman.Queues(queueID); err == nil {
			_ = UpdateEmbedWithComponents(s, i, renderQueuesEmbed(qs), componentsForQueues(qs))
		} else {
			// si no queda ninguna cola, deja UI mínima
			_ = UpdateEmbedWithComponents(s, i, renderQueuesEmbed(nil), componentsForQueues(nil))
		}
		return
	}

	switch customID {

	case "queue_join":
		if u == nil {
			_ = SendEphemeral(s, i, "⚠️ No pudimos identificarte.")
			return
		}

		_, err := qman.JoinAny(queueID, u.ID, u.Username, defaultCapacity)
		if err != nil {
			if errors.Is(err, queue.ErrAlreadyIn) {
				// ✅ No-op: no toques el embed; solo avisa efímero
				_ = SendEphemeral(s, i, "Ya estás en una cola.")
				return
			}
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}

		if qs, e := qman.Queues(queueID); e == nil {
			_ = UpdateEmbedWithComponents(s, i, renderQueuesEmbed(qs), componentsForQueues(qs))
		}
		return

	case "queue_leave":
		if u == nil {
			_ = SendEphemeral(s, i, "⚠️ No pudimos identificarte.")
			return
		}
		if _, err := qman.LeaveAny(queueID, u.ID); err != nil {
			switch {
			case errors.Is(err, queue.ErrNotIn):
				_ = SendEphemeral(s, i, "⚠️ No estás en ninguna cola.")
			default:
				_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			}
			return
		}
		if qs, e := qman.Queues(queueID); e == nil {
			_ = UpdateEmbedWithComponents(s, i, renderQueuesEmbed(qs), componentsForQueues(qs))
		}
		return
	}
}

// // DEBUG: llena la(s) cola(s) con N jugadores falsos
func seedDemoPlayers(n int, channelID string) {
	for i := 1; i <= n; i++ {
		id := fmt.Sprintf("seed-%02d", i)   // IDs sintéticos
		name := fmt.Sprintf("Seed %02d", i) // nombres visibles
		_, _ = qman.JoinAny(channelID, id, name, defaultCapacity)
	}
}
