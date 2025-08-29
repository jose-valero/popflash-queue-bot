package popflash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	c := &http.Client{Timeout: 6 * 1e9} // 6s
	return &Client{Base: base, Key: key, HTTP: c}
}

// Mantiene la firma actual que usa el resto del código.
func (c *Client) MatchCard(ctx context.Context, id string) (ui.MatchCard, error) {
	var zero ui.MatchCard

	url := fmt.Sprintf("%s/api/rest/match/%s", c.Base, id)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "popflash-queue-bot/1.0")
	if c.Key != "" {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return zero, fmt.Errorf("popflash GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return zero, fmt.Errorf("popflash GET %s -> %d", url, resp.StatusCode)
	}

	var payload getMatchResp
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return zero, err
	}

	return toUIMatchCard(payload.Match), nil
}
