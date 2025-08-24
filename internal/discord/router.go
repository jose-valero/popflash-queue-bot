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

// ------------------- Router principal ----------------
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

		SetQueueOpen(queueID, true)

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

		if !IsQueueOpen(queueID) {
			_ = SendEphemeral(s, i, "🔒 La cola está cerrada. Espera el próximo **match started**.")
			return
		}

		// exige voz si VOICE_REQUIRE_TO_JOIN=true
		if voiceRequireToJoin && !isUserInAllowedVoice(s, i.GuildID, u.ID) {
			_ = SendEphemeral(s, i, "🔇 Debes estar en un canal de voz en XCG 🔥 ó PopFlash Matches para unirte.")
			return
		}

		if _, err := qman.JoinAny(queueID, u.ID, u.Username, defaultCapacity); err != nil {
			if errors.Is(err, queue.ErrAlreadyIn) {
				_ = SendEphemeral(s, i, "Ya estás en una cola.")
				return
			}
			_ = SendEphemeral(s, i, "⚠️ "+err.Error())
			return
		}

		// refrescá el embed público
		if qs, e := qman.Queues(queueID); e == nil {
			_ = EditQueueMessage(s, queueID, renderQueuesEmbed(qs), componentsForQueues(qs))
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
			_ = UpdateEmbedWithComponents(s, i, renderQueuesEmbed(nil), componentsForQueues(nil))
		}
		return
	}

	// Select "queue_kick" — kickea al jugador elegido (requiere permisos)
	if customID == "queue_kick" {
		if !requirePrivileged(s, i) {
			return
		}

		vals := i.MessageComponentData().Values
		if len(vals) == 0 {
			_ = SendEphemeral(s, i, "⚠️ Selección inválida.")
			return
		}
		uid := strings.TrimPrefix(vals[0], "uid:")

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

		if qs, err := qman.Queues(queueID); err == nil {
			_ = UpdateEmbedWithComponents(s, i, renderQueuesEmbed(qs), componentsForQueues(qs))
		} else {
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

		if !IsQueueOpen(queueID) {
			_ = SendEphemeral(s, i, "🔒 La cola está cerrada. Espera el próximo **match started**.")
			return
		}

		// exige voz si VOICE_REQUIRE_TO_JOIN=true
		if voiceRequireToJoin && !isUserInAllowedVoice(s, i.GuildID, u.ID) {
			_ = SendEphemeral(s, i, "🔇 Debes estar en un canal de voz en XCG 🔥 ó PopFlash Matches para unirte.")
			return
		}

		_, err := qman.JoinAny(queueID, u.ID, u.Username, defaultCapacity)
		if err != nil {
			if errors.Is(err, queue.ErrAlreadyIn) {
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

	// Desconectado de voz → programa kick por AUSENTE
	if newChID == "" {
		scheduleAbsentKick(s, guildID, targetChannelID, userID)
		cancelAFK(userID)
		return
	}

	// AFK primero (siempre)
	if isAfkChannel(s, guildID, newChID) {
		cancelAbsent(userID)
		scheduleAFKKick(s, guildID, targetChannelID, userID)
		return
	}

	// En voz permitida → cancela ausente/AFK según corresponda
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

	// Voz NO permitida → trata como ausente
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

// Quita hasta N jugadores del frente de la primera cola de este canal.
func removeTopNFromQueue(channelID string, n int) {
	if n <= 0 {
		return
	}
	qs, err := qman.Queues(channelID)
	if err != nil || len(qs) == 0 || len(qs[0].Players) == 0 {
		return
	}
	players := qs[0].Players
	limit := n
	if len(players) < n {
		limit = len(players)
	}
	for i := 0; i < limit; i++ {
		_, _ = qman.LeaveAny(channelID, players[i].ID)
	}
}

// Publica lista de jugadores actuales en la primera cola
func announceQueueSnapshot(s *discordgo.Session, channelID string) {
	qs, err := qman.Queues(channelID)
	if err != nil || len(qs) == 0 {
		_, _ = s.ChannelMessageSend(channelID, "📋 No hay colas activas.")
		return
	}
	players := qs[0].Players
	if len(players) == 0 {
		_, _ = s.ChannelMessageSend(channelID, "📋 La cola está vacía.")
		return
	}
	var b strings.Builder
	b.WriteString("📋 **Jugadores en la cola:**\n")
	for i, p := range players {
		fmt.Fprintf(&b, "%2d) %s\n", i+1, p.Username) // o usa <@%s> con p.ID si querés mencionar
	}
	_, _ = s.ChannelMessageSend(channelID, b.String())
}

// Lógica de “match started”
func onMatchStarted(s *discordgo.Session, guildID, channelID string) {
	// Si veníamos de cola cerrada, purga a los primeros N (capacidad actual)
	prevOpen := IsQueueOpen(channelID)
	if !prevOpen {
		removeTopNFromQueue(channelID, defaultCapacity)
	}

	// Asegura cola y ábrela
	_, _ = qman.EnsureFirstQueue(channelID, "Queue #1", defaultCapacity)
	SetQueueOpen(channelID, true)

	// Render UI
	if qs, err := qman.Queues(channelID); err == nil {
		_ = PublishOrEditQueueMessage(s, channelID, renderQueuesEmbed(qs), componentsForQueues(qs))
	}

	_, _ = s.ChannelMessageSend(channelID, "🟢 **Cola abierta** — usa los botones para unirte/dejar la cola.")
}

// Lógica de “match finished”
func onMatchFinished(s *discordgo.Session, guildID, channelID string) {
	SetQueueOpen(channelID, false)
	announceQueueSnapshot(s, channelID)

	// Re-render UI (mismo embed; joins bloqueados por lógica)
	if qs, err := qman.Queues(channelID); err == nil {
		_ = PublishOrEditQueueMessage(s, channelID, renderQueuesEmbed(qs), componentsForQueues(qs))
	}

	_, _ = s.ChannelMessageSend(channelID, "🔴 **Cola cerrada** — esperando el próximo **match started**.")
}

// Devuelve si el mensaje (texto o embeds) contiene los triggers
func detectPFTriggers(msg *discordgo.Message) (started, finished bool) {
	txt := strings.ToLower(msg.Content)
	if strings.Contains(txt, "match started") {
		started = true
	}
	if strings.Contains(txt, "match finished") {
		finished = true
	}

	for _, e := range msg.Embeds {
		t := strings.ToLower(e.Title + " " + e.Description)
		if strings.Contains(t, "match started") {
			started = true
		}
		if strings.Contains(t, "match finished") {
			finished = true
		}
	}
	return
}

// Escucha mensajes del canal de scrims para “match started/finished”
func HandleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignora SOLO tus propios mensajes para evitar loops
	if m.Author != nil && m.Author.ID == s.State.User.ID {
		return
	}
	// Limitar al canal configurado (scrims)
	target := pfAnnounceChannelID
	if target == "" {
		target = m.ChannelID
	}
	if m.ChannelID != target {
		return
	}

	started, finished := detectPFTriggers(m.Message)
	if started {
		onMatchStarted(s, m.GuildID, m.ChannelID)
		return
	}
	if finished {
		onMatchFinished(s, m.GuildID, m.ChannelID)
		return
	}

	// (Si también vas a parsear links PopFlash, agregalo acá)
}

func HandleMessageUpdate(s *discordgo.Session, ev *discordgo.MessageUpdate) {
	if ev.Author != nil && ev.Author.ID == s.State.User.ID {
		return
	}
	target := pfAnnounceChannelID
	if target == "" {
		target = ev.ChannelID
	}
	if ev.ChannelID != target {
		return
	}

	// ev.Message puede venir parcial, pero normalmente incluye embeds/título
	started, finished := detectPFTriggers(ev.Message)
	if started {
		onMatchStarted(s, ev.GuildID, ev.ChannelID)
		return
	}
	if finished {
		onMatchFinished(s, ev.GuildID, ev.ChannelID)
		return
	}
}
