// internal/app/subscriber.go
package app

import (
	"errors" // ⬅️ importa esto
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
		cancels = append(cancels, events.Subscribe(func(_ events.MatchStarted) {
			channelID := b.Cfg.QueueChannelID
			if recentlyHandled("start:"+channelID, 3*time.Second) {
				return
			}

			// Asegurá que exista Q#1 antes de tocar la lista
			_, _ = qman.EnsureFirstQueue(channelID, "Queue #1", defaultCapacity)

			// Opcional: pop de Q#1
			if popped, _ := qman.PopFromFirst(channelID, defaultCapacity); len(popped) > 0 {
				log.Printf("[bus] auto-pop %d from Queue#1 in %s", len(popped), channelID)
			}

			// Marcar abierta ANTES de render
			SetQueueOpen(channelID, true)

			// Snapshot SIEMPRE válido (fallback si ErrNotFound)
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
					ui.RenderQueuesEmbed(qs, true),
					ui.ComponentsForQueues(qs, true), // ⬅️ botón habilitado
				)
			}

			log.Printf("[bus] MatchStarted → queue OPEN in %s", channelID)
		}))

		// ---------- MATCH FINISHED ----------
		cancels = append(cancels, events.Subscribe(func(_ events.MatchFinished) {
			channelID := b.Cfg.QueueChannelID
			if recentlyHandled("finish:"+channelID, 3*time.Second) {
				return
			}
			SetQueueOpen(channelID, false)

			// Snapshot con fallback mínimo
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
					ui.RenderQueuesEmbed(qs, false),
					ui.ComponentsForQueues(qs, false), // ⬅️ botón deshabilitado
				)
			}

			log.Printf("[bus] MatchFinished → queue CLOSED in %s", channelID)
		}))

		log.Printf("[bus] subscribers registered (once)")
		log.Printf("[bus] counts: MatchStarted=%d MatchFinished=%d",
			events.Count[events.MatchStarted](),
			events.Count[events.MatchFinished](),
		)

		subsCancel = func() {
			for _, c := range cancels {
				c()
			}
		}
		log.Printf("[bus] subscribers registered (once)")
	})

	return subsCancel
}
