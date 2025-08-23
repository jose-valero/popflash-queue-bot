// internal/queue/manager_test.go
package queue

import (
	"fmt"
	"sync"
	"testing"
)

func TestCreateQueueAndGet(t *testing.T) {
	m := NewManager()
	q, err := m.CreateQueue("q1", "Match Queue", 3)
	if err != nil {
		t.Fatalf("unexpected error creating queue: %v", err)
	}
	if q.ID != "q1" || q.Name != "Match Queue" || q.Capacity != 3 {
		t.Fatalf("bad queue data: %+v", q)
	}

	got, err := m.GetQueue("q1")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if got.ID != "q1" || got.Name != "Match Queue" || got.Capacity != 3 {
		t.Fatalf("bad snapshot: %+v", got)
	}
}

func TestCreateQueueTwice(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateQueue("q1", "A", 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := m.CreateQueue("q1", "B", 2); err == nil || err != ErrExists {
		t.Fatalf("expected ErrExists, got %v", err)
	}
}

func TestJoinLeaveHappyPath(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateQueue("q1", "Q", 5)

	if err := m.JoinQueue("q1", "u1", "alice"); err != nil {
		t.Fatalf("join u1: %v", err)
	}
	if err := m.JoinQueue("q1", "u2", "bob"); err != nil {
		t.Fatalf("join u2: %v", err)
	}

	q, _ := m.GetQueue("q1")
	if len(q.Players) != 2 || q.Players[0].ID != "u1" || q.Players[1].ID != "u2" {
		t.Fatalf("unexpected players: %+v", q.Players)
	}

	if err := m.LeaveQueue("q1", "u1"); err != nil {
		t.Fatalf("leave u1: %v", err)
	}
	q, _ = m.GetQueue("q1")
	if len(q.Players) != 1 || q.Players[0].ID != "u2" {
		t.Fatalf("unexpected players after leave: %+v", q.Players)
	}
}

func TestJoinAlreadyIn(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateQueue("q1", "Q", 5)
	_ = m.JoinQueue("q1", "u1", "alice")

	if err := m.JoinQueue("q1", "u1", "alice"); err != ErrAlreadyIn {
		t.Fatalf("expected ErrAlreadyIn, got %v", err)
	}
}

func TestLeaveNotIn(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateQueue("q1", "Q", 5)

	if err := m.LeaveQueue("q1", "uX"); err != ErrNotIn {
		t.Fatalf("expected ErrNotIn, got %v", err)
	}
}

func TestJoinFull(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateQueue("q1", "Q", 2)

	_ = m.JoinQueue("q1", "u1", "alice")
	_ = m.JoinQueue("q1", "u2", "bob")

	if err := m.JoinQueue("q1", "u3", "carol"); err != ErrFull {
		t.Fatalf("expected ErrFull, got %v", err)
	}
}

func TestDeleteQueue(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateQueue("q1", "Q", 2)
	m.DeleteQueue("q1")

	if _, err := m.GetQueue("q1"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSnapshotIsImmutable(t *testing.T) {
	m := NewManager()
	_, _ = m.CreateQueue("q1", "Q", 5)
	_ = m.JoinQueue("q1", "u1", "alice")

	snap, _ := m.GetQueue("q1")
	if len(snap.Players) != 1 {
		t.Fatalf("bad snapshot: %+v", snap.Players)
	}
	// Mutar el snapshot NO debe afectar el estado real
	snap.Players = nil

	real, _ := m.GetQueue("q1")
	if len(real.Players) != 1 {
		t.Fatalf("manager state was affected by snapshot mutation")
	}
}

func TestConcurrentJoinsRespectsCapacity(t *testing.T) {
	m := NewManager()
	const capQ = 5
	_, _ = m.CreateQueue("q1", "Q", capQ)

	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("u%d", n) // IDs únicos
			if err := m.JoinQueue("q1", id, id); err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	q, _ := m.GetQueue("q1")
	if success != capQ || len(q.Players) != capQ {
		t.Fatalf("expected %d success and %d players, got success=%d len=%d", capQ, capQ, success, len(q.Players))
	}
}
