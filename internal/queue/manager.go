package queue

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type Manager struct {
	mu     sync.RWMutex
	byChan map[string]*channelQueues // channelID -> colas en ese canal
}

type channelQueues struct {
	Queues []*Queue // índice 0-based; mostramos 1-based en UI
}

var (
	ErrExists    = errors.New("queue already exists")
	ErrNotFound  = errors.New("queue not found")
	ErrFull      = errors.New("queue is full")
	ErrAlreadyIn = errors.New("already in queue")
	ErrNotIn     = errors.New("player not in queue")
)

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

// Asegura que exista la Queue #1
func (m *Manager) EnsureFirstQueue(channelID, name string, capacity int) (*Queue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cq := m.getOrCreateChannel(channelID)
	if len(cq.Queues) == 0 {
		if capacity <= 0 {
			return nil, errors.New("invalid capacity")
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

// Snapshot de todas las colas del canal
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

// JoinAny: mete al jugador en la primera cola con espacio; si no hay, crea otra.
func (m *Manager) JoinAny(channelID, playerID, username string, capacity int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cq := m.getOrCreateChannel(channelID)

	// ya está en alguna?
	for idx, q := range cq.Queues {
		for _, p := range q.Players {
			if p.ID == playerID {
				return idx + 1, ErrAlreadyIn
			}
		}
	}

	// primera con hueco
	for idx, q := range cq.Queues {
		if len(q.Players) < q.Capacity {
			q.Players = append(q.Players, Player{
				ID:       playerID,
				Username: username,
				JoinedAt: time.Now().UTC(),
			})
			return idx + 1, nil
		}
	}

	// crear nueva
	if capacity <= 0 {
		capacity = 5
	}
	newIdx := len(cq.Queues) + 1
	q := &Queue{
		ID:        fmt.Sprintf("%s:%d", channelID, newIdx),
		Name:      fmt.Sprintf("Queue #%d", newIdx),
		Players:   []Player{{ID: playerID, Username: username, JoinedAt: time.Now().UTC()}},
		CreatedAt: time.Now().UTC(),
		Capacity:  capacity,
	}
	cq.Queues = append(cq.Queues, q)
	return newIdx, nil
}

// LeaveAny: saca al jugador de la cola en que esté y re-balancea promoviendo.
func (m *Manager) LeaveAny(channelID, playerID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cq, ok := m.byChan[channelID]
	if !ok {
		return 0, ErrNotFound
	}

	foundIdx := -1
	for idx, q := range cq.Queues {
		for i, p := range q.Players {
			if p.ID == playerID {
				q.Players = append(q.Players[:i], q.Players[i+1:]...)
				foundIdx = idx
				break
			}
		}
		if foundIdx >= 0 {
			break
		}
	}
	if foundIdx < 0 {
		return 0, ErrNotIn
	}

	m.rebalance(cq) // PROMOVER: Q2->Q1, Q3->Q2, ...
	return foundIdx + 1, nil
}

// ResetAt: vacía SOLO la cola idx (1-based) y rebalancea.
func (m *Manager) ResetAt(channelID string, idx int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cq, ok := m.byChan[channelID]
	if !ok || idx <= 0 || idx > len(cq.Queues) {
		return ErrNotFound
	}
	cq.Queues[idx-1].Players = []Player{}
	m.rebalance(cq)
	return nil
}

// DeleteAt: elimina SOLO la cola idx (1-based) y rebalancea.
func (m *Manager) DeleteAt(channelID string, idx int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cq, ok := m.byChan[channelID]
	if !ok || idx <= 0 || idx > len(cq.Queues) {
		return ErrNotFound
	}
	cq.Queues = append(cq.Queues[:idx-1], cq.Queues[idx:]...)
	m.rebalance(cq)
	return nil
}

// ===== helpers =====

func (m *Manager) rebalance(cq *channelQueues) {
	// Para cada cola j, mientras haya hueco, traer del primer jugador de j+1
	for j := 0; j < len(cq.Queues)-1; j++ {
		for len(cq.Queues[j].Players) < cq.Queues[j].Capacity && len(cq.Queues[j+1].Players) > 0 {
			// pop front de j+1
			p := cq.Queues[j+1].Players[0]
			cq.Queues[j+1].Players = cq.Queues[j+1].Players[1:]
			// push a j (preservando orden)
			cq.Queues[j].Players = append(cq.Queues[j].Players, p)
		}
	}
	// Eliminar colas vacías al final (dejar al menos una)
	for len(cq.Queues) > 1 && len(cq.Queues[len(cq.Queues)-1].Players) == 0 {
		cq.Queues = cq.Queues[:len(cq.Queues)-1]
	}
}

func snapshot(q *Queue) *Queue {
	cp := *q
	cp.Players = append([]Player(nil), q.Players...)
	return &cp
}
