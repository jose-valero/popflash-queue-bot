// Package queue - manager.go
// Concurrency-safe manager of per-channel, multi-queue sets.
package queue

import (
	"fmt"
	"sync"
	"time"
)

// Manager keeps per-channel queue sets behind a RWMutex.
type Manager struct {
	mu     sync.RWMutex
	byChan map[string]*channelQueues // channelID -> queues in that channel
}

type channelQueues struct {
	Queues []*Queue // 0-based indexes; UI can render 1-based
}

// NewManager constructs an empty Manager.
func NewManager() *Manager {
	return &Manager{byChan: make(map[string]*channelQueues)}
}

func (m *Manager) getOrCreateChannel(channelID string) *channelQueues {
	cq, ok := m.byChan[channelID]
	if !ok {
		cq = &channelQueues{Queues: []*Queue{}}
		m.byChan[channelID] = cq
	}
	return cq
}

// EnsureFirstQueue makes sure Queue #1 exists in the given channel.
// If created, it uses the provided name and capacity; if capacity <= 0,
// it returns an error (keeps current semantics). Always returns a snapshot.
func (m *Manager) EnsureFirstQueue(channelID, name string, capacity int) (*Queue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cq := m.getOrCreateChannel(channelID)
	if len(cq.Queues) == 0 {
		if capacity <= 0 {
			return nil, qerr("invalid capacity")
		}
		q := &Queue{
			ID:        fmt.Sprintf("%s:%d", channelID, 1),
			Name:      name,
			Players:   []Player{},
			CreatedAt: time.Now().UTC(),
			Capacity:  capacity,
		}
		cq.Queues = append(cq.Queues, q)
		return snapshot(q), nil
	}
	return snapshot(cq.Queues[0]), nil
}

// Queues returns a deep-copy snapshot of all queues in a channel.
func (m *Manager) Queues(channelID string) ([]*Queue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cq, ok := m.byChan[channelID]
	if !ok || len(cq.Queues) == 0 {
		return nil, ErrNotFound
	}
	out := make([]*Queue, 0, len(cq.Queues))
	for _, q := range cq.Queues {
		out = append(out, snapshot(q))
	}
	return out, nil
}

// JoinAny appends the player to the first queue with space; otherwise it
// creates a new queue and joins there. Returns the 1-based queue index.
// If the player is already in any queue, returns ErrAlreadyIn.
func (m *Manager) JoinAny(channelID, playerID, username string, capacity int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cq := m.getOrCreateChannel(channelID)

	// Already present?
	for idx, q := range cq.Queues {
		for _, p := range q.Players {
			if p.ID == playerID {
				return idx + 1, ErrAlreadyIn
			}
		}
	}

	// First queue with room.
	now := time.Now().UTC()
	for idx, q := range cq.Queues {
		if len(q.Players) < q.Capacity {
			q.Players = append(q.Players, Player{ID: playerID, Username: username, JoinedAt: now})
			return idx + 1, nil
		}
	}

	// Create a new tail queue.
	if capacity <= 0 {
		capacity = 5
	}
	newIdx := len(cq.Queues) + 1
	q := &Queue{
		ID:        fmt.Sprintf("%s:%d", channelID, newIdx),
		Name:      fmt.Sprintf("Queue #%d", newIdx),
		Players:   []Player{{ID: playerID, Username: username, JoinedAt: now}},
		CreatedAt: now,
		Capacity:  capacity,
	}
	cq.Queues = append(cq.Queues, q)
	return newIdx, nil
}

// LeaveAny removes the player from whichever queue they are in, then
// rebalances forward. Returns the 1-based queue index from which the
// player was removed.
func (m *Manager) LeaveAny(channelID, playerID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cq, ok := m.byChan[channelID]
	if !ok {
		return 0, ErrNotFound
	}
	qi, pi := locatePlayer(cq.Queues, playerID)
	if qi < 0 {
		return 0, ErrNotIn
	}

	q := cq.Queues[qi]
	q.Players = append(q.Players[:pi], q.Players[pi+1:]...)

	rebalanceForward(cq.Queues, qi)
	cq.Queues = pruneTrailingEmpty(cq.Queues)

	return qi + 1, nil
}

// ResetAt clears only the indicated queue (1-based index) and rebalances.
func (m *Manager) ResetAt(channelID string, idx int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cq, ok := m.byChan[channelID]
	if !ok || idx <= 0 || idx > len(cq.Queues) {
		return ErrNotFound
	}
	cq.Queues[idx-1].Players = cq.Queues[idx-1].Players[:0]
	rebalanceForward(cq.Queues, idx-1)
	cq.Queues = pruneTrailingEmpty(cq.Queues)
	return nil
}

// DeleteAt removes the indicated queue (1-based index) and then rebalances.
// If the last queues become empty, trailing empties are pruned.
func (m *Manager) DeleteAt(channelID string, idx int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cq, ok := m.byChan[channelID]
	if !ok || idx <= 0 || idx > len(cq.Queues) {
		return ErrNotFound
	}
	cq.Queues = append(cq.Queues[:idx-1], cq.Queues[idx:]...)
	// After removing an entire queue, rebalancing from the previous index
	// keeps earlier queues as full as possible.
	rebalanceForward(cq.Queues, max(0, idx-2))
	cq.Queues = pruneTrailingEmpty(cq.Queues)
	return nil
}

// tiny util
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
