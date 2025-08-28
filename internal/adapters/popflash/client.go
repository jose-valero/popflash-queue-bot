// internal/adapters/popflash/client.go
package popflash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jose-valero/popflash-queue-bot/internal/ui"
)

type Client struct {
	Base string
	Key  string
	HTTP *http.Client
}

func New(base, key string) *Client {
	if base == "" {
		base = "https://api.popflash.site"
	}
	c := &http.Client{Timeout: 6 * time.Second}
	return &Client{Base: base, Key: key, HTTP: c}
}

type getMatchResp struct {
	Match struct {
		ID         int     `json:"id"`
		Map        *string `json:"map"`
		Datacenter *int    `json:"datacenter"`
		CreatedAt  *string `json:"created_at"`
		Status     *string `json:"status"` // NUEVO (opcional)
		Score1     *int    `json:"score1"` // NUEVO
		Score2     *int    `json:"score2"` // NUEVO
		Users      []struct {
			Team *int `json:"team"` // 1 o 2
			User *struct {
				Name *string `json:"name"`
			} `json:"user"`
		} `json:"users_matches"`
	} `json:"match"`
}

func dcLabel(dc *int) string {
	if dc == nil {
		return "—"
	}
	return fmt.Sprintf("DC %d", *dc)
}

func parseTime(ts *string) time.Time {
	if ts == nil || *ts == "" {
		return time.Time{}
	}
	s := *ts
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000000-07",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (c *Client) MatchCard(ctx context.Context, id string) (ui.MatchCard, error) {
	var card ui.MatchCard

	url := fmt.Sprintf("%s/api/rest/match/%s", c.Base, id)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if c.Key != "" {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return card, fmt.Errorf("popflash GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return card, fmt.Errorf("popflash GET %s -> %d", url, resp.StatusCode)
	}

	var payload getMatchResp
	if err = json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return card, err
	}
	m := payload.Match

	var t1, t2 []string
	for _, um := range m.Users {
		name := ""
		if um.User != nil && um.User.Name != nil {
			name = *um.User.Name
		}
		if name == "" {
			continue
		}
		if um.Team != nil && *um.Team == 2 {
			t2 = append(t2, name)
		} else {
			t1 = append(t1, name)
		}
	}

	mp := "—"
	if m.Map != nil && *m.Map != "" {
		mp = *m.Map
	}

	card = ui.MatchCard{
		ID:      id,
		Map:     mp,
		Region:  dcLabel(m.Datacenter),
		Started: parseTime(m.CreatedAt),
		Team1:   t1,
		Team2:   t2,
		Score1:  m.Score1, // ← NUEVO
		Score2:  m.Score2, // ← NUEVO
	}
	return card, nil
}
