package discord

import (
	"log"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

var (
	queueMsgIDs sync.Map // channelID -> messageID
	chLocks     sync.Map // channelID -> *sync.Mutex
)

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

func chanLock(channelID string) *sync.Mutex {
	v, _ := chLocks.LoadOrStore(channelID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// Heurística mínima para detectar "nuestro" mensaje de UI existente.
// Usa el título de tu embed (ajústalo si lo cambias).
func looksLikeQueueUI(m *discordgo.Message) bool {
	if len(m.Embeds) == 0 {
		return false
	}
	t := m.Embeds[0].Title
	return strings.Contains(strings.ToLower(t), "ellos la llevan") // <-- tu marca
}

// Busca un mensaje anterior de UI en el canal.
func findExistingQueueMessage(s *discordgo.Session, channelID string) (string, bool) {
	msgs, err := s.ChannelMessages(channelID, 50, "", "", "")
	if err != nil {
		return "", false
	}

	botID := ""
	if s.State != nil && s.State.User != nil {
		botID = s.State.User.ID
	}
	for _, m := range msgs {
		if m == nil || len(m.Embeds) == 0 {
			continue
		}
		if botID != "" && (m.Author == nil || m.Author.ID != botID) {
			continue
		}
		if looksLikeQueueUI(m) {
			return m.ID, true
		}
	}
	return "", false
}

// PublishOrEditQueueMessage: protegido por lock por canal + doble chequeo.
// - Si tenemos ID: edita.
// - Si no tenemos, intenta recuperar del historial y edita.
// - Si no existe, crea uno nuevo y recuerda su ID.
func PublishOrEditQueueMessage(s *discordgo.Session, channelID string, emb *discordgo.MessageEmbed, comps []discordgo.MessageComponent) error {
	mu := chanLock(channelID)
	mu.Lock()
	defer mu.Unlock()

	if _, ok := getQueueMessageID(channelID); ok {
		log.Printf("[publisher] EDIT (remembered) ch=%s", channelID)
		return EditQueueMessage(s, channelID, emb, comps)
	}
	if id, ok := findExistingQueueMessage(s, channelID); ok {
		log.Printf("[publisher] EDIT (rehydrated id=%s) ch=%s", id, channelID)
		SetQueueMessageID(channelID, id)
		return EditQueueMessage(s, channelID, emb, comps)
	}
	msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{emb}, Components: comps})
	if err != nil {
		return err
	}
	if msg != nil {
		log.Printf("[publisher] CREATE id=%s ch=%s", msg.ID, channelID)
		SetQueueMessageID(channelID, msg.ID)
	}
	return nil
}

func EditQueueMessage(s *discordgo.Session, channelID string, emb *discordgo.MessageEmbed, comps []discordgo.MessageComponent) error {
	msgID, ok := getQueueMessageID(channelID)
	if !ok {
		return nil // no recordado aún (lo resuelve PublishOrEdit)
	}
	embeds := []*discordgo.MessageEmbed{emb}
	compsCopy := comps
	_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    channelID,
		ID:         msgID,
		Embeds:     &embeds,
		Components: &compsCopy,
	})
	if err != nil {
		// Si el mensaje ya no existe (10008), olvidamos el ID y dejamos que PublishOrEdit lo recree
		if re, ok := err.(*discordgo.RESTError); ok && re.Message != nil && re.Message.Code == 10008 {
			queueMsgIDs.Delete(channelID)
			return PublishOrEditQueueMessage(s, channelID, emb, comps)
		}
	}
	return err
}
