package queue

import (
	"errors"
	"sync"
	"time"
)

type Manager struct {
	queues map[string]*Queue
	mu     sync.RWMutex
}

var (
	ErrExists    = errors.New("queue already exists")
	ErrNotFound  = errors.New("queue not found")
	ErrFull      = errors.New("queue is full")
	ErrAlreadyIn = errors.New("already in queue")
	ErrNotIn     = errors.New("player not in queue")
)

func NewManager() *Manager {
	return &Manager{queues: make(map[string]*Queue)}
}

func (m *Manager) CreateQueue(id, name string, capacity int) (*Queue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.queues[id]; exists {
		return nil, ErrExists
	}
	if capacity <= 0 {
		return nil, errors.New("invalid capacity")
	}
	q := &Queue{
		ID:        id,
		Name:      name,
		Players:   []Player{},
		CreatedAt: time.Now().UTC(),
		Capacity:  capacity,
	}
	m.queues[id] = q
	return q, nil
}

func (m *Manager) JoinQueue(queueID, playerID, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, exists := m.queues[queueID]
	if !exists {
		return ErrNotFound
	}
	if len(q.Players) >= q.Capacity {
		return ErrFull
	}
	for _, p := range q.Players {
		if p.ID == playerID {
			return ErrAlreadyIn
		}
	}
	q.Players = append(q.Players, Player{
		ID:       playerID,
		Username: username,
		JoinedAt: time.Now().UTC(),
	})
	return nil
}

func (m *Manager) LeaveQueue(queueID, playerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, exists := m.queues[queueID]
	if !exists {
		return ErrNotFound
	}
	for i, p := range q.Players {
		if p.ID == playerID {
			q.Players = append(q.Players[:i], q.Players[i+1:]...)
			return nil
		}
	}
	return ErrNotIn
}

func (m *Manager) GetQueue(queueID string) (*Queue, error) {
	m.mu.RLock()
	q, ok := m.queues[queueID]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return snapshot(q), nil
}

func (m *Manager) DeleteQueue(queueID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.queues, queueID)
}

func (m *Manager) ResetQueue(queueID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, ok := m.queues[queueID]
	if !ok {
		return ErrNotFound
	}

	q.Players = []Player{}
	return nil
}

func snapshot(q *Queue) *Queue {
	cp := *q
	cp.Players = append([]Player(nil), q.Players...)
	return &cp
}
