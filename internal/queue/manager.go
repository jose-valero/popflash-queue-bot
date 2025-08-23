package queue

import (
	"errors"
	"sync"
	"time"
)

type Manager struct {
	queues map[string]*Queue
	mu     sync.Mutex
}

// manager of queue
func NewManager() *Manager {

	return &Manager{
		queues: make(map[string]*Queue),
	}
}

// create a queueu
func (m *Manager) CreateQueue(id, name string, capacity int) (*Queue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.queues[id]; exists {
		return nil, errors.New("already exits a queue with the id")
	}

	q := &Queue{
		ID:        id,
		Name:      name,
		Players:   []Player{},
		CreatedAt: time.Now(),
		Capacity:  capacity,
	}

	m.queues[id] = q
	return q, nil
}

// join to queue
func (m *Manager) JoinQueue(queueID, playerID, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, exists := m.queues[queueID]
	if !exists {
		return errors.New("queue not found")
	}

	if len(q.Players) >= q.Capacity {
		return errors.New("full queue")
	}

	// Verificar si ya está en la cola
	for _, p := range q.Players {
		if p.ID == playerID {
			return errors.New("you already in queue")
		}
	}

	q.Players = append(q.Players, Player{
		ID:       playerID,
		Username: username,
		JoinedAt: time.Now(),
	})

	return nil
}

// leave a queue
func (m *Manager) LeaveQueue(queueID, playerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	q, exists := m.queues[queueID]
	if !exists {
		return errors.New("queue not found")
	}

	for i, p := range q.Players {
		if p.ID == playerID {
			q.Players = append(q.Players[:i], q.Players[i+1:]...)
			return nil
		}
	}
	return errors.New("player not found in queue")
}

// get a queue state
func (m *Manager) GetQueue(queueID string) (*Queue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, exists := m.queues[queueID]
	if !exists {
		return nil, errors.New("queue not found")
	}

	return q, nil
}
