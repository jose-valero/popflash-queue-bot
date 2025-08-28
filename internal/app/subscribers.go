// internal/app/subscribers.go
package app

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	d "github.com/jose-valero/popflash-queue-bot/internal/adapters/discord"
	events "github.com/jose-valero/popflash-queue-bot/internal/domain/events"
	"github.com/jose-valero/popflash-queue-bot/internal/queue"
	"github.com/jose-valero/popflash-queue-bot/internal/ui"
)

var subsOnce sync.Once
var subsCancel func() = func() {}
var handled sync.Map

func recentlyHandled(key string, ttl time.Duration) bool {
	now := time.Now()
	if v, ok := handled.Load(key); ok {
		if now.Sub(v.(time.Time)) < ttl {
			return true
		}
	}
	handled.Store(key, now)
	return false
}

func (b *Bot) StartEventSubscribers() func() {
	subsOnce.Do(func() {
		var cancels []func()

		// ---------- MATCH STARTED ----------
		cancels = append(cancels, events.Subscribe(func(ev events.MatchStarted) {
			channelID := b.Cfg.QueueChannelID
			if recentlyHandled("start:"+channelID, 3*time.Second) {
				return
			}

			// Asegura Q#1
			_, _ = qman.EnsureFirstQueue(channelID, "Queue #1", defaultCapacity)

			// Opcional: pop de Q#1 al comenzar
			if popped, _ := qman.PopFromFirst(channelID, defaultCapacity); len(popped) > 0 {
				log.Printf("[bus] auto-pop %d from Queue#1 in %s", len(popped), channelID)
			}

			// Si hay cliente PF y tenemos MatchID, hidrata y guarda card activa
			if ev.MatchID != "" {
				if b.PF != nil {
					if card, err := b.PF.MatchCard(context.Background(), ev.MatchID); err == nil {
						ActivePut(card)
						log.Printf("[bus] active put match=%s map=%s region=%s", ev.MatchID, card.Map, card.Region)

					} else {
						log.Printf("[bus] PF MatchCard(%s) error: %v — using minimal card", ev.MatchID, err)
						ActivePut(ui.MatchCard{ID: ev.MatchID, Started: time.Now()})
						log.Printf("[bus] active put match=%s map=%s region=%s", ev.MatchID, card.Map, card.Region)

					}
				} else {
					ActivePut(ui.MatchCard{ID: ev.MatchID, Started: time.Now()})

				}
			}

			// Abrimos la cola
			SetQueueOpen(channelID, true)

			// Snapshot con fallback
			qs, err := qman.Queues(channelID)
			if errors.Is(err, queue.ErrNotFound) {
				if q, e2 := qman.EnsureFirstQueue(channelID, "Queue #1", defaultCapacity); e2 == nil && q != nil {
					qs = []*queue.Queue{q}
					err = nil
				}
			}
			if err == nil {
				_ = d.PublishOrEditQueueMessage(
					b.Sess, channelID,
					ui.RenderQueuesEmbed(qs, true, cardsOrNil(b)),
					ui.ComponentsForQueues(qs, true),
				)
			}

			log.Printf("[bus] MatchStarted → queue OPEN in %s", channelID)
		}))

		// ---------- MATCH FINISHED ----------
		cancels = append(cancels, events.Subscribe(func(ev events.MatchFinished) {
			channelID := b.Cfg.QueueChannelID
			if recentlyHandled("finish:"+channelID, 3*time.Second) {
				return
			}

			// Quita la partida de “activas”
			if ev.MatchID != "" {
				ActiveRemove(ev.MatchID)
			}

			// Si aún quedan partidas activas, mantenemos la cola abierta
			open := ActiveCount() > 0
			SetQueueOpen(channelID, open)

			qs, err := qman.Queues(channelID)
			if errors.Is(err, queue.ErrNotFound) {
				if q, e2 := qman.EnsureFirstQueue(channelID, "Queue #1", defaultCapacity); e2 == nil && q != nil {
					qs = []*queue.Queue{q}
					err = nil
				}
			}
			if err == nil {
				_ = d.PublishOrEditQueueMessage(
					b.Sess, channelID,
					ui.RenderQueuesEmbed(qs, open, cardsOrNil(b)),
					ui.ComponentsForQueues(qs, open),
				)
			}

			log.Printf("[bus] MatchFinished → queue %s in %s",
				map[bool]string{true: "OPEN", false: "CLOSED"}[open], channelID)
		}))

		log.Printf("[bus] subscribers registered (once)")

		subsCancel = func() {
			for _, c := range cancels {
				c()
			}
		}
	})

	return subsCancel
}
func cardsOrNil(b *Bot) []ui.MatchCard {
	if b.Cfg != nil && b.Cfg.FFActiveMatchesUI {
		return ActiveList()
	}
	return nil
}
