// Voice policy helpers: read env, decide if a channel/user is allowed,
// cache last voice channel, and expose a tiny handler to keep that cache fresh.

package discord

import (
	"os"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

var (
	voiceOnce sync.Once

	voiceRequireToJoin     bool
	allowedCategoryIDs     map[string]struct{}
	allowedCategoryNames   map[string]struct{}
	allowedChannelPrefixes []string
	afkChannelOverride     string

	// (guildID:userID) -> last voice channelID
	lastVoice sync.Map
)

func loadVoicePolicyFromEnv() {
	// VOICE_REQUIRE_TO_JOIN = "true"/"1" enables the check in your join handlers
	v := strings.TrimSpace(os.Getenv("VOICE_REQUIRE_TO_JOIN"))
	voiceRequireToJoin = v == "1" || strings.EqualFold(v, "true")

	// Allow-list by category IDs (comma separated)
	allowedCategoryIDs = make(map[string]struct{})
	for _, id := range strings.Split(os.Getenv("VOICE_ALLOWED_CATEGORY_IDS"), ",") {
		id = strings.Trim(strings.TrimSpace(id), `"'`)
		if id != "" {
			allowedCategoryIDs[id] = struct{}{}
		}
	}

	// Allow-list by category NAMES (comma separated; case-insensitive)
	allowedCategoryNames = make(map[string]struct{})
	for _, n := range strings.Split(os.Getenv("VOICE_ALLOWED_CATEGORY_NAMES"), ",") {
		n = strings.ToLower(strings.Trim(strings.TrimSpace(n), `"'`))
		if n != "" {
			allowedCategoryNames[n] = struct{}{}
		}
	}

	// Optional allow-list by channel NAME prefix (comma separated)
	for _, p := range strings.Split(os.Getenv("VOICE_ALLOWED_CHANNEL_PREFIXES"), ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			allowedChannelPrefixes = append(allowedChannelPrefixes, p)
		}
	}

	// Optional explicit AFK channel override
	afkChannelOverride = strings.TrimSpace(os.Getenv("AFK_CHANNEL_ID"))
}

// VoiceRequireToJoin reports whether /join should enforce voice policy.
func VoiceRequireToJoin() bool {
	voiceOnce.Do(loadVoicePolicyFromEnv)
	return voiceRequireToJoin
}

// TrackVoiceState is a lightweight handler to keep the "last voice channel"
// cache updated. Register it in app.Bot.RegisterHandlers().
func TrackVoiceState(_ *discordgo.Session, ev *discordgo.VoiceStateUpdate) {
	voiceOnce.Do(loadVoicePolicyFromEnv)
	if ev == nil || ev.VoiceState == nil {
		return
	}
	vs := ev.VoiceState
	lastVoice.Store(vs.GuildID+":"+vs.UserID, vs.ChannelID)
}

func getLastVoice(guildID, userID string) (string, bool) {
	k := guildID + ":" + userID
	v, ok := lastVoice.Load(k)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// IsUserInAllowedVoice returns true if the user is currently in an allowed voice channel.
func IsUserInAllowedVoice(s *discordgo.Session, guildID, userID string) bool {
	voiceOnce.Do(loadVoicePolicyFromEnv)

	// Prefer our cache (fast, works across events)
	if chID, ok := getLastVoice(guildID, userID); ok && chID != "" {
		return ChannelAllowedByCategory(s, chID)
	}

	// Fallback to discordgo state
	vs, err := s.State.VoiceState(guildID, userID)
	if err != nil || vs == nil || vs.ChannelID == "" {
		return false
	}
	return ChannelAllowedByCategory(s, vs.ChannelID)
}

// ChannelAllowedByCategory applies the configured allow-lists.
// If no allow-lists are configured, ALL channels are considered allowed.
func ChannelAllowedByCategory(s *discordgo.Session, channelID string) bool {
	voiceOnce.Do(loadVoicePolicyFromEnv)
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

	// 1) Allow by channel name prefix (if configured)
	if len(allowedChannelPrefixes) > 0 {
		name := strings.ToLower(ch.Name)
		for _, pref := range allowedChannelPrefixes {
			if strings.HasPrefix(name, pref) {
				return true
			}
		}
	}

	// 2) Allow by category ID
	if ch.ParentID != "" {
		if _, ok := allowedCategoryIDs[ch.ParentID]; ok {
			return true
		}

		// 3) Allow by category NAME (case-insensitive)
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

	// 4) If no allow-lists configured, default to ALLOW
	return len(allowedCategoryIDs) == 0 && len(allowedCategoryNames) == 0 && len(allowedChannelPrefixes) == 0
}

// IsAFKChannel reports whether channelID is the guild AFK channel.
// Honors AFK_CHANNEL_ID override if set.
func IsAFKChannel(s *discordgo.Session, guildID, channelID string) bool {
	voiceOnce.Do(loadVoicePolicyFromEnv)
	if channelID == "" {
		return false
	}
	if afkChannelOverride != "" {
		return channelID == afkChannelOverride
	}
	g, _ := s.State.Guild(guildID)
	if g == nil {
		g, _ = s.Guild(guildID)
	}
	return g != nil && g.AfkChannelID != "" && g.AfkChannelID == channelID
}
