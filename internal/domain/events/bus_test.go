package events

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type E1 struct{ A int }
type E2 struct{ S string }

func TestBus_SubscribePublish_TypeIsolation(t *testing.T) {
	var c1 int32

	cancel := Subscribe(func(ev E1) {
		atomic.AddInt32(&c1, int32(ev.A))
	})
	defer cancel()

	Publish(E1{A: 1})
	Publish(E1{A: 2})
	Publish(E2{S: "noop"}) // no debe afectar

	if got := atomic.LoadInt32(&c1); got != 3 {
		t.Fatalf("want 3, got %d", got)
	}
}

func TestBus_Cancel_Unsubscribe(t *testing.T) {
	var hits int32

	cancel := Subscribe(func(E1) {
		atomic.AddInt32(&hits, 1)
	})
	cancel() // desuscribir antes de publicar

	Publish(E1{A: 1})
	time.Sleep(10 * time.Millisecond)

	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("want 0 after cancel, got %d", got)
	}
}

func TestBus_Concurrency_NoRaces(t *testing.T) {
	var hits int32

	cancel := Subscribe(func(E1) {
		atomic.AddInt32(&hits, 1)
	})
	defer cancel()

	const G = 50
	const N = 100
	var wg sync.WaitGroup
	wg.Add(G)
	for g := 0; g < G; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < N; i++ {
				Publish(E1{A: 1})
			}
		}()
	}
	wg.Wait()

	want := int32(G * N)
	if got := atomic.LoadInt32(&hits); got != want {
		t.Fatalf("want %d, got %d", want, got)
	}
}
