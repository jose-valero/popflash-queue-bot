// Package popflash provides a thin HTTP client for the PopFlash REST API.
// It only models the fields we currently need; additional JSON is ignored.
package popflash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Client is a minimal PopFlash API client.
type Client struct {
	BaseURL  string
	ClanSlug string
	http     *http.Client
}

// NewClientFromEnv builds a client using POPFLASH_BASE_URL and POPFLASH_CLAN_SLUG.
// Defaults BaseURL to https://api.popflash.site if not set.
func NewClientFromEnv() *Client {
	base := os.Getenv("POPFLASH_BASE_URL")
	if base == "" {
		base = "https://api.popflash.site"
	}
	slug := os.Getenv("POPFLASH_CLAN_SLUG")

	// Reasonable HTTP defaults.
	h := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
		},
	}
	return &Client{
		BaseURL:  base,
		ClanSlug: slug,
		http:     h,
	}
}

// ---- API types (partial; extend as needed) ----

type Match struct {
	ID     int64  `json:"id"`
	Status int    `json:"status"` // discover semantics empirically (e.g., 1=in_progress, 2=finished)
	Map    string `json:"map"`

	// Optional summary fields for nice finish embeds:
	Team1 *Team `json:"team1,omitempty"`
	Team2 *Team `json:"team2,omitempty"`
	// Some endpoints might nest teams/players differently; keep types flexible.
}

type Team struct {
	Name    string   `json:"name"`
	Score   int      `json:"score"`
	Players []Player `json:"players,omitempty"`
}

type Player struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// MatchSummary is a lighter view used by list endpoints.
type MatchSummary struct {
	ID        int64     `json:"id"`
	Status    int       `json:"status"`
	Map       string    `json:"map"`
	CreatedAt time.Time `json:"created_at"`
}

// ---- Endpoints ----

// GetMatch returns the match detail for the given match id.
func (c *Client) GetMatch(ctx context.Context, id int64) (*Match, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/rest/match/%d", c.BaseURL, id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("popflash: %s", resp.Status)
	}
	var m Match
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ListClanMatches lists recent matches for the configured clan slug.
// Use small limits (e.g., 5–10) to reconcile state on boot.
func (c *Client) ListClanMatches(ctx context.Context, limit, offset int) ([]MatchSummary, error) {
	if c.ClanSlug == "" {
		return nil, errors.New("popflash: missing ClanSlug (POPFLASH_CLAN_SLUG)")
	}
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/rest/clan/%s/matches?limit=%s&offset=%s",
			c.BaseURL, c.ClanSlug, strconv.Itoa(limit), strconv.Itoa(offset)),
		nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("popflash: %s", resp.Status)
	}
	var out []MatchSummary
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
