package queue

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func invariant(t *testing.T, qs []*Queue) {
	// 1) capacidad
	for _, q := range qs {
		if len(q.Players) > q.Capacity {
			t.Fatalf("capacity exceeded: %d/%d", len(q.Players), q.Capacity)
		}
	}
	// 2) sin duplicados
	seen := map[string]bool{}
	for _, q := range qs {
		for _, p := range q.Players {
			if seen[p.ID] {
				t.Fatalf("duplicate player %s", p.ID)
			}
			seen[p.ID] = true
		}
	}
}

func TestJoinLeaveBasic(t *testing.T) {
	m := NewManager()
	ch := "c1"
	if _, err := m.EnsureFirstQueue(ch, "Q1", 5); err != nil {
		t.Fatal(err)
	}

	// join 7 -> 5/2 (dos colas)
	for i := 0; i < 7; i++ {
		_, err := m.JoinAny(ch, // channel
			string(rune('A'+i)), // playerID
			"u", 5)
		if err != nil {
			t.Fatal(err)
		}
	}
	qs, _ := m.Queues(ch)
	invariant(t, qs)

	// leave uno, rebalancea hacia delante
	_, _ = m.LeaveAny(ch, "A")
	qs, _ = m.Queues(ch)
	invariant(t, qs)
}

func TestRaceRandomOps(t *testing.T) {
	m := NewManager()
	ch := "c"
	_, _ = m.EnsureFirstQueue(ch, "Q1", 5)
	rand.Seed(time.Now().UnixNano())

	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				id := "P" + string(rune('A'+rand.Intn(26)))
				if rand.Intn(2) == 0 {
					_, _ = m.JoinAny(ch, id, id, 5)
				} else {
					_, _ = m.LeaveAny(ch, id)
				}
			}
		}(g)
	}
	wg.Wait()

	qs, _ := m.Queues(ch)
	invariant(t, qs)
}
