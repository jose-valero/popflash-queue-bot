package discord

import (
	"log"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/jose-valero/popflash-queue-bot/internal/domain/events"
)

var announceChannelID string // set by app at startup

// SetAnnounceChannel configures which text channel we listen to.
func SetAnnounceChannel(id string) { announceChannelID = id }

// ---- NUEVO: patrones tolerantes a "Match 123 started/finished"
var (
	reStarted  = regexp.MustCompile(`(?i)\bmatch(?:\s*[#:]?\s*\d+)?\s*started\b`)
	reFinished = regexp.MustCompile(`(?i)\bmatch(?:\s*[#:]?\s*\d+)?\s*finished\b`)
)

func looksStarted(s string) bool  { return reStarted.MatchString(s) }
func looksFinished(s string) bool { return reFinished.MatchString(s) }

// detectPFTriggers inspects text + embeds and returns (started, finished).
func detectPFTriggers(msg *discordgo.Message) (started, finished bool) {
	if msg == nil {
		return
	}

	// 1) Texto plano del mensaje (a veces PopFlash incluye una línea además del embed)
	if looksStarted(msg.Content) {
		started = true
	}
	if looksFinished(msg.Content) {
		finished = true
	}

	// 2) Títulos/descripciones/fields del embed
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
	}

	if started || finished {
		log.Printf("[announcer] detected started=%t finished=%t", started, finished)
	}
	return
}

// HandleMessageCreate emits bus events when announcements arrive.
func HandleMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if announceChannelID == "" || m.ChannelID != announceChannelID {
		return
	}

	// DEPURACIÓN: imprime títulos de embeds recibidos
	if len(m.Embeds) > 0 {
		var titles []string
		for _, e := range m.Embeds {
			titles = append(titles, e.Title)
		}
		log.Printf("[announcer] embeds=%d titles=%q", len(m.Embeds), titles)
	} else {
		log.Printf("[announcer] content=%q (no embeds)", m.Content)
	}

	started, finished := detectPFTriggers(m.Message)
	if started {
		log.Printf("[announcer] detected MATCH STARTED")
		events.Publish(events.MatchStarted{
			GuildID:   m.GuildID,
			ChannelID: m.ChannelID,
			MessageID: m.ID,
		})
	}
	if finished {
		log.Printf("[announcer] detected MATCH FINISHED")
		events.Publish(events.MatchFinished{
			GuildID:   m.GuildID,
			ChannelID: m.ChannelID,
			MessageID: m.ID,
		})
	}
}

// HandleMessageUpdate also catches edits that add/remove the trigger words.
func HandleMessageUpdate(_ *discordgo.Session, ev *discordgo.MessageUpdate) {
	if announceChannelID == "" || ev.ChannelID != announceChannelID {
		return
	}

	// DEPURACIÓN: imprime títulos de embeds en updates
	if ev.Message != nil && len(ev.Message.Embeds) > 0 {
		var titles []string
		for _, e := range ev.Message.Embeds {
			titles = append(titles, e.Title)
		}
		log.Printf("[announcer][update] embeds=%d titles=%q", len(ev.Message.Embeds), titles)
	}

	started, finished := detectPFTriggers(ev.Message)
	if started {
		log.Printf("[announcer][update] detected MATCH STARTED")
		events.Publish(events.MatchStarted{
			GuildID:   ev.GuildID,
			ChannelID: ev.ChannelID,
			MessageID: ev.ID,
		})
	}
	if finished {
		log.Printf("[announcer][update] detected MATCH FINISHED")
		events.Publish(events.MatchFinished{
			GuildID:   ev.GuildID,
			ChannelID: ev.ChannelID,
			MessageID: ev.ID,
		})
	}
}
