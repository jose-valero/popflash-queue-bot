// internal/app/score_poller.go
package app

import (
	"context"
	"log"
	"sync"
	"time"

	d "github.com/jose-valero/popflash-queue-bot/internal/adapters/discord"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
	"github.com/jose-valero/popflash-queue-bot/internal/ui"
)

var pollOnce sync.Once

func (b *Bot) StartScorePoller() {
	pollOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				// No hay partidas o no hay cliente: nada que hacer.
				if b == nil || b.PF == nil || ActiveCount() == 0 {
					continue
				}

				ids := make([]string, 0, ActiveCount())
				for _, c := range ActiveList() {
					ids = append(ids, c.ID)
				}
				if len(ids) == 0 {
					continue
				}

				changed := false
				for _, id := range ids {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					card, err := b.PF.MatchCard(ctx, id)
					cancel()
					if err != nil {
						log.Printf("[poll] match %s err: %v", id, err)
						continue
					}
					// Guardamos (si no cambió, sobrescribe igual; está OK)
					ActivePut(card)
					changed = true

					// micro-pausa entre requests, por cortesía
					time.Sleep(1200 * time.Millisecond)
				}

				// Si actualizamos algo, re-renderizamos el embed público
				if changed {
					ch := b.Cfg.QueueChannelID
					qs, err := qman.Queues(ch)
					if err != nil && err != queue.ErrNotFound {
						continue
					}
					if err == queue.ErrNotFound {
						if q, e2 := qman.EnsureFirstQueue(ch, "Queue #1", defaultCapacity); e2 == nil && q != nil {
							qs = []*queue.Queue{q}
						}
					}
					_ = d.PublishOrEditQueueMessage(
						b.Sess, ch,
						ui.RenderQueuesEmbed(qs, IsQueueOpen(ch), ActiveList()),
						ui.ComponentsForQueues(qs, IsQueueOpen(ch)),
					)
				}
			}
		}()
	})
}
