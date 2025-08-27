package app

import (
	"log"
	"sync"
	"time"

	d "github.com/jose-valero/popflash-queue-bot/internal/adapters/discord"
	events "github.com/jose-valero/popflash-queue-bot/internal/domain/events"
	"github.com/jose-valero/popflash-queue-bot/internal/ui"
)

var subsOnce sync.Once            // <— NUEVO
var subsCancel func() = func() {} // idempotente
var handled sync.Map              // key -> time

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
	subsOnce.Do(func() { // <— evita doble registro aunque lo llamen dos veces
		var cancels []func()

		// PopFlash → match started
		cancels = append(cancels, events.Subscribe(func(_ events.MatchStarted) {
			channelID := b.Cfg.QueueChannelID
			if recentlyHandled("start:"+channelID, 3*time.Second) {
				return
			}
			_, _ = qman.EnsureFirstQueue(channelID, "Queue #1", defaultCapacity)
			SetQueueOpen(channelID, true)

			if qs, err := qman.Queues(channelID); err == nil {
				_ = d.PublishOrEditQueueMessage(
					b.Sess, channelID,
					ui.RenderQueuesEmbed(qs, IsQueueOpen(channelID)),
					ui.ComponentsForQueues(qs, true),
				)
			}
			log.Printf("[bus] MatchStarted → queue OPEN in %s", channelID)
		}))

		// PopFlash → match finished
		cancels = append(cancels, events.Subscribe(func(_ events.MatchFinished) {
			channelID := b.Cfg.QueueChannelID
			if recentlyHandled("start:"+channelID, 3*time.Second) {
				return
			}
			SetQueueOpen(channelID, false)

			if qs, err := qman.Queues(channelID); err == nil {
				_ = d.PublishOrEditQueueMessage(
					b.Sess, channelID,
					ui.RenderQueuesEmbed(qs, IsQueueOpen(channelID)),
					ui.ComponentsForQueues(qs, false),
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
