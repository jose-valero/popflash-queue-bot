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
	} // ejemplo
	return &Client{Base: base, Key: key, HTTP: &http.Client{Timeout: 5 * time.Second}}
}

func (c *Client) MatchCard(ctx context.Context, id string) (ui.MatchCard, error) {
	var card ui.MatchCard
	// TODO: ajusta endpoint real
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/matches/%s", c.Base, id), nil)
	if c.Key != "" {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return card, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return card, fmt.Errorf("popflash %d", resp.StatusCode)
	}
	// TODO: mapear JSON real al MatchCard
	// var m apiMatch
	// _ = json.NewDecoder(resp.Body).Decode(&m)
	// card = mapApiToCard(m)
	_ = json.NewDecoder(resp.Body).Decode(&struct{}{}) // placeholder
	// mientras no esté la API, devolvemos un mock defensivo:
	card = ui.MatchCard{
		ID: id, Map: "de_inferno", Region: "Atlanta",
		Started: time.Now().Add(-27 * time.Minute),
		Team1:   []string{"A", "B", "C"},
		Team2:   []string{"D", "E", "F"},
	}
	return card, nil
}
