package discord

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const ephemeralFlag = 1 << 6

var adminRoleSet = map[string]struct{}{}

var afkChannelOverride string

var allowedCategoryIDs = map[string]struct{}{}   // ya lo tenías como allowedVoiceCategories
var allowedCategoryNames = map[string]struct{}{} // por nombre de categoría (lowercase)
var allowedChannelPrefixes []string              // prefijos de nombre de canal de voz (lowercase)

var (
	// Reglas de VOZ
	voiceRequireToJoin = false

	// Ventanas de gracia
	absentGrace = 15 * time.Minute
	afkGrace    = 1 * time.Minute
)

// Guarda el messageID del embed público por canal para poder editar sin interacción
var queueMsgIDs sync.Map // channelID -> messageID

// Cache: última voz conocida por (guild,user)
var lastVoice sync.Map

func voiceKey(guildID, userID string) string { return guildID + ":" + userID }
func setLastVoice(guildID, userID, channelID string) {
	lastVoice.Store(voiceKey(guildID, userID), channelID)
}
func getLastVoice(guildID, userID string) (string, bool) {
	v, ok := lastVoice.Load(voiceKey(guildID, userID))
	if !ok {
		return "", false
	}
	return v.(string), true
}

func init() {
	// Lee ADMIN_ROLE_IDS al iniciar el proceso
	for _, id := range strings.Split(os.Getenv("ADMIN_ROLE_IDS"), ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			adminRoleSet[id] = struct{}{}
		}
	}

	// Reglas de voz
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("VOICE_REQUIRE_TO_JOIN"))); v == "true" || v == "1" {
		voiceRequireToJoin = true
	}
	fillSetFromCSV(allowedCategoryIDs, os.Getenv("VOICE_ALLOWED_CATEGORY_IDS"))

	// Nombres de categorías
	for _, raw := range strings.Split(os.Getenv("VOICE_ALLOWED_CATEGORY_NAMES"), ",") {
		n := strings.ToLower(strings.TrimSpace(raw))
		n = strings.Trim(n, `"'`)
		if n != "" {
			allowedCategoryNames[n] = struct{}{}
		}
	}

	// Prefijos de nombre de canal
	for _, p := range strings.Split(os.Getenv("VOICE_ALLOWED_CHANNEL_PREFIXES"), ",") {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			allowedChannelPrefixes = append(allowedChannelPrefixes, p)
		}
	}

	afkChannelOverride = strings.TrimSpace(os.Getenv("AFK_CHANNEL_ID"))

	// Gracia
	if v := os.Getenv("ABSENT_GRACE_MIN"); v != "" {
		if m, err := strconv.Atoi(v); err == nil && m > 0 {
			absentGrace = time.Duration(m) * time.Minute
		}
	}
	if v := os.Getenv("AFK_GRACE_MIN"); v != "" {
		if m, err := strconv.Atoi(v); err == nil && m > 0 {
			afkGrace = time.Duration(m) * time.Minute
		}
	}
}

func fillSetFromCSV(dst map[string]struct{}, csv string) {
	for _, id := range strings.Split(csv, ",") {
		id = strings.TrimSpace(id)
		id = strings.Trim(id, `"'`)
		if id != "" {
			dst[id] = struct{}{}
		}
	}
}

// --------- permisos ---------
// Devuelve true si el usuario tiene Administrator o alguno de los roles configurados
func isPrivileged(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	// override por permiso Administrator
	if i.Member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}
	// match por roles configurados
	for _, r := range i.Member.Roles {
		if _, ok := adminRoleSet[r]; ok {
			return true
		}
	}
	return false
}

// Atajo: responde efímero y corta si no tiene permisos
func requirePrivileged(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if isPrivileged(i) {
		return true
	}
	_ = SendEphemeral(s, i, "⛔ No tienes permiso para esta acción.")
	return false
}

// --------- VOZ: helpers ---------

// ¿el canal de voz pertenece a una categoría permitida?
func channelAllowedByCategory(s *discordgo.Session, channelID string) bool {
	if channelID == "" {
		return false
	}
	ch, err := s.State.Channel(channelID)
	if err != nil || ch == nil {
		ch, _ = s.Channel(channelID) // REST fallback
	}
	if ch == nil {
		return false
	}

	// 1) Permitir por prefijo de NOMBRE de canal (opcional)
	if len(allowedChannelPrefixes) > 0 {
		name := strings.ToLower(ch.Name)
		for _, pref := range allowedChannelPrefixes {
			if strings.HasPrefix(name, pref) {
				return true
			}
		}
	}

	// 2) Permitir por ID de CATEGORÍA
	if ch.ParentID != "" {
		if _, ok := allowedCategoryIDs[ch.ParentID]; ok {
			return true
		}
		// 3) Permitir por NOMBRE de CATEGORÍA
		if len(allowedCategoryNames) > 0 {
			var catName string
			if cat, _ := s.State.Channel(ch.ParentID); cat != nil {
				catName = cat.Name
			} else if cat2, _ := s.Channel(ch.ParentID); cat2 != nil {
				catName = cat2.Name
			}
			if _, ok := allowedCategoryNames[strings.ToLower(strings.TrimSpace(catName))]; ok {
				return true
			}
		}
	}

	// 4) Si no hay ninguna regla configurada, todo vale. Si hay reglas, no.
	return len(allowedCategoryIDs) == 0 && len(allowedCategoryNames) == 0 && len(allowedChannelPrefixes) == 0
}

func isUserInAllowedVoice(s *discordgo.Session, guildID, userID string) bool {
	// 1) Primero mira nuestro cache
	if chID, ok := getLastVoice(guildID, userID); ok && chID != "" {
		return channelAllowedByCategory(s, chID)
	}
	// 2) Fallback al caché de discordgo (puede estar vacío si el bot arrancó tarde)
	vs, err := s.State.VoiceState(guildID, userID)
	if err != nil || vs == nil || vs.ChannelID == "" {
		return false
	}
	return channelAllowedByCategory(s, vs.ChannelID)
}

// --------- Embed público: guardar/editar mensaje ---------

func SetQueueMessageID(channelID, messageID string) {
	if channelID != "" && messageID != "" {
		queueMsgIDs.Store(channelID, messageID)
	}
}

func getQueueMessageID(channelID string) (string, bool) {
	v, ok := queueMsgIDs.Load(channelID)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// Edita el mensaje público (embed + componentes) sin interacción
func EditQueueMessage(s *discordgo.Session, channelID string, emb *discordgo.MessageEmbed, comps []discordgo.MessageComponent) error {
	msgID, ok := getQueueMessageID(channelID)
	if !ok {
		return nil // aún no tenemos el messageID
	}

	embeds := []*discordgo.MessageEmbed{emb} // necesitamos un slice para tomarle la dirección
	// OJO: tomar la dirección del slice (no de un literal)
	compsCopy := comps

	_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    channelID,
		ID:         msgID,
		Embeds:     &embeds,    // <- *([]*MessageEmbed)
		Components: &compsCopy, // <- *([]MessageComponent)
	})
	return err
}

func isAfkChannel(s *discordgo.Session, guildID, channelID string) bool {
	if channelID == "" {
		return false
	}
	// Si configuraste un AFK explícito, usalo
	if afkChannelOverride != "" {
		return channelID == afkChannelOverride
	}
	// Si no, usa el AFK del guild
	g, _ := s.State.Guild(guildID)
	if g == nil {
		g, _ = s.Guild(guildID)
	}
	return g != nil && g.AfkChannelID != "" && g.AfkChannelID == channelID
}

// --------- helpers de respuesta de texto ---------

func SendResponse(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg},
	})
	if err != nil {
		log.Printf("SendResponse error: %v", err)
	}
	return err
}

func SendResponseWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, msg string, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    msg,
			Components: comps,
		},
	})
	if err != nil {
		log.Printf("SendResponseWithComponents error: %v", err)
	}
	return err
}

func SendEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   ephemeralFlag,
		},
	})
	if err != nil {
		log.Printf("SendEphemeral error: %v", err)
	}
	return err
}

func SendEphemeralWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, msg string, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    msg,
			Components: comps,
			Flags:      ephemeralFlag,
		},
	})
	if err != nil {
		log.Printf("SendEphemeralWithComponents error: %v", err)
	}
	return err
}

// --------- helpers de EMBED ---------

func SendEmbedWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, emb *discordgo.MessageEmbed, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{emb},
			Components: comps,
		},
	})
	if err != nil {
		log.Printf("SendEmbedWithComponents error: %v", err)
	}
	return err
}

func UpdateEmbedWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, emb *discordgo.MessageEmbed, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{emb},
			Components: comps,
		},
	})
	if err != nil {
		log.Printf("UpdateEmbedWithComponents error: %v", err)
	}
	return err
}

func SendEphemeralEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, emb *discordgo.MessageEmbed) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{emb},
			Flags:  ephemeralFlag,
		},
	})
	if err != nil {
		log.Printf("SendEphemeralEmbed error: %v", err)
	}
	return err
}

// --------- update simple de texto (por completitud) ---------

func UpdateMessageWithComponents(s *discordgo.Session, i *discordgo.InteractionCreate, content string, comps []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: comps,
		},
	})
	if err != nil {
		log.Printf("UpdateMessageWithComponents error: %v", err)
	}
	return err
}

// --------- util de usuario ---------

func userOf(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	return i.User
}

func safeName(u *discordgo.User) string {
	if u == nil {
		return "unknown"
	}
	return u.Username
}

func notifyUserKicked(s *discordgo.Session, userID, reason string) {
	if targetChannelID == "" {
		return
	}
	_, err := s.ChannelMessageSendComplex(targetChannelID, &discordgo.MessageSend{
		Content: fmt.Sprintf("⏱️ <@%s> te saqué de la cola: %s", userID, reason),
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Users: []string{userID}, // solo menciona al expulsado
		},
	})
	if err != nil {
		log.Printf("notifyUserKicked error: %v", err)
	}
}

// --- DEBUG helpers ---
// func mapKeys(m map[string]struct{}) []string {
// 	out := make([]string, 0, len(m))
// 	for k := range m {
// 		out = append(out, k)
// 	}
// 	return out
// }

// func safe(s string) string {
// 	if s == "" {
// 		return "?"
// 	}
// 	return s
// }

// func debugVoiceSnapshot(s *discordgo.Session, guildID, userID string) (voiceID, voiceName, parentID, parentName string) {
// 	// Primero cache propio
// 	if chID, ok := getLastVoice(guildID, userID); ok && chID != "" {
// 		voiceID = chID
// 	} else {
// 		// Fallback a state de discordgo
// 		if vs, _ := s.State.VoiceState(guildID, userID); vs != nil {
// 			voiceID = vs.ChannelID
// 		}
// 	}

// 	if voiceID != "" {
// 		if ch, _ := s.State.Channel(voiceID); ch != nil {
// 			voiceName = ch.Name
// 			parentID = ch.ParentID
// 		} else if ch2, _ := s.Channel(voiceID); ch2 != nil {
// 			voiceName = ch2.Name
// 			parentID = ch2.ParentID
// 		}
// 		if parentID != "" {
// 			if cat, _ := s.State.Channel(parentID); cat != nil {
// 				parentName = cat.Name
// 			} else if cat2, _ := s.Channel(parentID); cat2 != nil {
// 				parentName = cat2.Name
// 			}
// 		}
// 	}
// 	return
// }
