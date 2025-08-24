package discord

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
)

var (
	qman            = queue.NewManager()
	targetChannelID string
	defaultCapacity = 5
)

// Timers por usuario para ausente/AFK
type userTimer struct {
	t         *time.Timer
	channelID string
}

var absentTimers sync.Map // userID -> *userTimer
var afkTimers sync.Map    // userID -> *userTimer

func cancelTimer(m *sync.Map, userID string) {
	if v, ok := m.Load(userID); ok {
		if ut := v.(*userTimer); ut != nil && ut.t != nil {
			ut.t.Stop()
		}
		m.Delete(userID)
	}
}

func cancelAbsent(userID string) { cancelTimer(&absentTimers, userID) }
func cancelAFK(userID string)    { cancelTimer(&afkTimers, userID) }

func scheduleAbsentKick(s *discordgo.Session, guildID, channelID, userID string) {
	cancelAbsent(userID)
	ut := &userTimer{channelID: channelID}
	ut.t = time.AfterFunc(absentGrace, func() {
		// si volvió a voz permitida, no kickear
		if isUserInAllowedVoice(s, guildID, userID) {
			return
		}
		_, _ = qman.LeaveAny(channelID, userID)

		notifyUserKicked(s, userID, fmt.Sprintf("no estabas en un canal de voz permitido por %v.", absentGrace))

		if qs, err := qman.Queues(channelID); err == nil {
			_ = EditQueueMessage(s, channelID, renderQueuesEmbed(qs), componentsForQueues(qs))
		}
		absentTimers.Delete(userID)
	})
	absentTimers.Store(userID, ut)
}

func scheduleAFKKick(s *discordgo.Session, guildID, channelID, userID string) {
	cancelAFK(userID)
	ut := &userTimer{channelID: channelID}
	ut.t = time.AfterFunc(afkGrace, func() {
		g, _ := s.State.Guild(guildID)
		if g == nil {
			g, _ = s.Guild(guildID)
		}
		vs, _ := s.State.VoiceState(guildID, userID)
		if g == nil || vs == nil || !isAfkChannel(s, guildID, vs.ChannelID) {
			return
		}
		_, _ = qman.LeaveAny(channelID, userID)

		// if _, err := qman.LeaveAny(channelID, userID); err == nil {
		// 	notifyUserKicked(s, userID,
		// 		fmt.Sprintf("estuviste en el canal AFK por %d min.", int(afkGrace/time.Minute)))
		// }
		notifyUserKicked(s, userID, fmt.Sprintf("estuviste en el canal AFK por %v.", afkGrace))

		if qs, err := qman.Queues(channelID); err == nil {
			_ = EditQueueMessage(s, channelID, renderQueuesEmbed(qs), componentsForQueues(qs))
		}
		afkTimers.Delete(userID)
	})
	afkTimers.Store(userID, ut)
}

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
		// 1) Crear (o asegurar) la primera cola de este canal
		if _, err := qman.EnsureFirstQueue(queueID, "Queue #1", defaultCapacity); err != nil {
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}

		// (opcional) DEMO: prellenar para probar
		// seedDemoPlayers(11, queueID)

		// 2) Ack diferido para no timeoutear la interacción
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			log.Printf("defer error: %v", err)
			return
		}

		// 3) Renderizar el estado actual y editar la respuesta ORIGINAL
		qs, _ := qman.Queues(queueID)
		embeds := []*discordgo.MessageEmbed{renderQueuesEmbed(qs)}
		comps := componentsForQueues(qs)

		msg, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &embeds, // puntero al slice
			Components: &comps,  // puntero al slice
		})
		if err != nil {
			log.Printf("edit original response error: %v", err)
			_ = SendEphemeral(s, i, "⚠️ No pude renderizar la cola.")
			return
		}

		// 4) Guardar el messageID para futuras ediciones sin interacción
		if msg != nil {
			SetQueueMessageID(queueID, msg.ID)
		}
		return

	case "joinqueue":
		u := userOf(i)
		if u == nil {
			_ = SendEphemeral(s, i, "⚠️ No pudimos identificarte.")
			return
		}

		//👇 NUEVO: exige voz si VOICE_REQUIRE_TO_JOIN=true
		if voiceRequireToJoin && !isUserInAllowedVoice(s, i.GuildID, u.ID) {
			_ = SendEphemeral(s, i, "🔇 Debes estar en un canal de voz en XCG ó PopFlashMatches para unirte.")
			return
		}
		// 👇 NUEVO: exige voz si VOICE_REQUIRE_TO_JOIN=true + DEBUGGER
		// if voiceRequireToJoin && !isUserInAllowedVoice(s, i.GuildID, u.ID) {
		// 	vID, vName, pID, pName := debugVoiceSnapshot(s, i.GuildID, u.ID)
		// 	_ = SendEphemeral(s, i, fmt.Sprintf(
		// 		"🔇 Debes estar en un canal de voz permitido para unirte.\n\n"+
		// 			"**Detecté**\n• Voz: %s (`%s`)\n• Categoría: %s (`%s`)\n\n"+
		// 			"**Permitidos (.env)**\n• IDs categoría: %v\n• Nombres categoría: %v\n• Prefijos canal: %v",
		// 		safe(vName), safe(vID), safe(pName), safe(pID),
		// 		mapKeys(allowedCategoryIDs), mapKeys(allowedCategoryNames), allowedChannelPrefixes,
		// 	))
		// 	return
		// }

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

		// 👇 NUEVO: igual que arriba
		if voiceRequireToJoin && !isUserInAllowedVoice(s, i.GuildID, u.ID) {
			_ = SendEphemeral(s, i, "🔇 Debes estar en un canal de voz en XCG 🔥 ó PopFlashMatches para unirte.")
			return
		}

		// 👇 NUEVO: igual que arriba + debuger
		// if voiceRequireToJoin && !isUserInAllowedVoice(s, i.GuildID, u.ID) {
		// 	vID, vName, pID, pName := debugVoiceSnapshot(s, i.GuildID, u.ID)
		// 	_ = SendEphemeral(s, i, fmt.Sprintf(
		// 		"🔇 Debes estar en un canal de voz permitido para unirte.\n\n"+
		// 			"**Detecté**\n• Voz: %s (`%s`)\n• Categoría: %s (`%s`)\n\n"+
		// 			"**Permitidos (.env)**\n• IDs categoría: %v\n• Nombres categoría: %v\n• Prefijos canal: %v",
		// 		safe(vName), safe(vID), safe(pName), safe(pID),
		// 		mapKeys(allowedCategoryIDs), mapKeys(allowedCategoryNames), allowedChannelPrefixes,
		// 	))
		// 	return
		// }

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

// Observa cambios de voz y programa/cancela timers de absent/AFK.
// Nota: usamos targetChannelID para re-renderizar la UI del canal de colas.
func HandleVoiceStateUpdate(s *discordgo.Session, vsu *discordgo.VoiceStateUpdate) {
	vs := vsu.VoiceState
	if vs == nil {
		return
	}
	guildID := vs.GuildID
	userID := vs.UserID
	newChID := vs.ChannelID

	setLastVoice(guildID, userID, newChID)

	// Si no hay canal de colas configurado, no hacemos nada.
	if targetChannelID == "" {
		return
	}

	// 1) Desconectado de voz → programa kick por AUSENTE
	if newChID == "" {
		scheduleAbsentKick(s, guildID, targetChannelID, userID)
		cancelAFK(userID)
		return
	}

	// 1) AFK primero (siempre)
	if isAfkChannel(s, guildID, newChID) {
		cancelAbsent(userID)
		scheduleAFKKick(s, guildID, targetChannelID, userID)
		return
	}

	// 2) En alguna voz…
	//    - si es categoría permitida → cancela "ausente"
	//    - si además es AFK → programa kick por AFK; si no, cancela AFK
	if channelAllowedByCategory(s, newChID) {
		cancelAbsent(userID)

		g, _ := s.State.Guild(guildID)
		if g == nil {
			g, _ = s.Guild(guildID)
		}
		if g != nil && g.AfkChannelID != "" && newChID == g.AfkChannelID {
			scheduleAFKKick(s, guildID, targetChannelID, userID)
		} else {
			cancelAFK(userID)
		}
		return
	}

	// 3) En voz NO permitida → trata como ausente
	scheduleAbsentKick(s, guildID, targetChannelID, userID)
	cancelAFK(userID)
}

// Al entrar al guild, poblar lastVoice con los VoiceStates actuales
func HandleGuildCreate(s *discordgo.Session, ev *discordgo.GuildCreate) {
	if ev == nil || ev.Guild == nil {
		return
	}
	g := ev.Guild
	for _, vs := range g.VoiceStates {
		setLastVoice(g.ID, vs.UserID, vs.ChannelID)
	}
}

// // DEBUG: llena la(s) cola(s) con N jugadores falsos
// func seedDemoPlayers(n int, channelID string) {
// 	for i := 1; i <= n; i++ {
// 		id := fmt.Sprintf("seed-%02d", i)   // IDs sintéticos
// 		name := fmt.Sprintf("Seed %02d", i) // nombres visibles
// 		_, _ = qman.JoinAny(channelID, id, name, defaultCapacity)
// 	}
// }
