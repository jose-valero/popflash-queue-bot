// Package events - types.go
package events

// MatchStarted is emitted when a PopFlash match starts.
type MatchStarted struct {
	GuildID   string
	ChannelID string // channel where the announcement came from
	MessageID string
}

// MatchFinished is emitted when a PopFlash match finishes.
type MatchFinished struct {
	GuildID   string
	ChannelID string
	MessageID string
}
