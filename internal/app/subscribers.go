package app

import (
	"log"

	d "github.com/jose-valero/popflash-queue-bot/internal/adapters/discord"
	events "github.com/jose-valero/popflash-queue-bot/internal/domain/events"
	"github.com/jose-valero/popflash-queue-bot/internal/ui"
)

// StartEventSubscribers subscribes to the event bus and returns a cancel().
// Reacts to MatchStarted/MatchFinished by opening/closing the queue UI in the
// configured queue channel.
func (b *Bot) StartEventSubscribers() func() {
	var cancels []func()

	// PopFlash → match started
	cancels = append(cancels, events.Subscribe(func(_ events.MatchStarted) {
		channelID := b.Cfg.QueueChannelID

		// Ensure first queue exists and open it
		_, _ = qman.EnsureFirstQueue(channelID, "Queue #1", defaultCapacity)
		SetQueueOpen(channelID, true)

		// Render/update public UI
		if qs, err := qman.Queues(channelID); err == nil {
			_ = d.PublishOrEditQueueMessage(
				b.Sess, channelID,
				ui.RenderQueuesEmbed(qs),
				ui.ComponentsForQueues(qs, true),
			)
		}

		log.Printf("[bus] MatchStarted → queue OPEN in %s", channelID)
	}))

	// PopFlash → match finished
	cancels = append(cancels, events.Subscribe(func(_ events.MatchFinished) {
		channelID := b.Cfg.QueueChannelID
		SetQueueOpen(channelID, false)

		// Re-render (joins are blocked by app logic)
		if qs, err := qman.Queues(channelID); err == nil {
			_ = d.PublishOrEditQueueMessage(
				b.Sess, channelID,
				ui.RenderQueuesEmbed(qs),
				ui.ComponentsForQueues(qs, false),
			)
		}

		log.Printf("[bus] MatchFinished → queue CLOSED in %s", channelID)
	}))

	return func() {
		for _, c := range cancels {
			c()
		}
	}
}
