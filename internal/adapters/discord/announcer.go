package discord

import (
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/domain/events"
)

var announceChannelID string

func SetAnnounceChannel(id string) { announceChannelID = id }

// Patrones
var (
	reStarted  = regexp.MustCompile(`(?i)\bmatch(?:\s*[#:]?\s*\d+)?\s*started\b`)
	reFinished = regexp.MustCompile(`(?i)\bmatch(?:\s*[#:]?\s*\d+)?\s*finished\b`)
	reMatchID  = regexp.MustCompile(`(?i)\bmatch\s*[#:]?\s*(\d+)\b`)

	recent     sync.Map
	triggerTTL = 15 * time.Second
)

func allowOnce(key string) bool {
	now := time.Now()
	if v, ok := recent.Load(key); ok {
		if now.Sub(v.(time.Time)) < triggerTTL {
			return false
		}
	}
	recent.Store(key, now)
	return true
}

func looksStarted(s string) bool  { return reStarted.MatchString(s) }
func looksFinished(s string) bool { return reFinished.MatchString(s) }

func extractMatchIDFromStrings(strs ...string) string {
	for _, s := range strs {
		if s == "" {
			continue
		}
		if m := reMatchID.FindStringSubmatch(s); len(m) == 2 {
			return m[1] // número del match
		}
	}
	return ""
}

func detectPFTriggers(msg *discordgo.Message) (started, finished bool, matchID string) {
	if msg == nil {
		return
	}

	// 1) Texto plano
	if looksStarted(msg.Content) {
		started = true
	}
	if looksFinished(msg.Content) {
		finished = true
	}
	if matchID == "" {
		matchID = extractMatchIDFromStrings(msg.Content)
	}

	// 2) Embeds
	for _, e := range msg.Embeds {
		texts := []string{e.Title, e.Description}
		if e.Author != nil {
			texts = append(texts, e.Author.Name)
		}
		if e.Footer != nil {
			texts = append(texts, e.Footer.Text)
		}
		for _, f := range e.Fields {
			texts = append(texts, f.Name, f.Value)
		}

		for _, t := range texts {
			if !started && looksStarted(t) {
				started = true
			}
			if !finished && looksFinished(t) {
				finished = true
			}
		}
		if matchID == "" {
			matchID = extractMatchIDFromStrings(texts...)
		}
	}

	if started || finished {
		log.Printf("[announcer] detected started=%t finished=%t matchID=%q", started, finished, matchID)
	}
	return
}

func dedupeKey(id, matchID, suffix string) string {
	// Preferimos dedupe por matchID; si no hay, caemos al messageID
	if matchID != "" {
		return "match:" + matchID + "#" + suffix
	}
	return "msg:" + id + "#" + suffix
}

func HandleMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if announceChannelID == "" || m.ChannelID != announceChannelID {
		return
	}

	started, finished, mid := detectPFTriggers(m.Message)
	if started && allowOnce(dedupeKey(m.ID, mid, "start")) {
		log.Printf("[announcer] publish start key=%s", dedupeKey(m.ID, mid, "start"))
		events.Publish(events.MatchStarted{GuildID: m.GuildID, ChannelID: m.ChannelID, MessageID: m.ID, MatchID: mid})
	}
	if finished && allowOnce(dedupeKey(m.ID, mid, "finish")) {
		events.Publish(events.MatchFinished{GuildID: m.GuildID, ChannelID: m.ChannelID, MessageID: m.ID, MatchID: mid})
	}
}

func HandleMessageUpdate(_ *discordgo.Session, ev *discordgo.MessageUpdate) {
	if announceChannelID == "" || ev.ChannelID != announceChannelID {
		return
	}

	started, finished, mid := detectPFTriggers(ev.Message)
	if started && allowOnce(dedupeKey(ev.ID, mid, "start")) {
		events.Publish(events.MatchStarted{GuildID: ev.GuildID, ChannelID: ev.ChannelID, MessageID: ev.ID, MatchID: mid})
	}
	if finished && allowOnce(dedupeKey(ev.ID, mid, "finish")) {
		events.Publish(events.MatchFinished{GuildID: ev.GuildID, ChannelID: ev.ChannelID, MessageID: ev.ID, MatchID: mid})
	}
}
