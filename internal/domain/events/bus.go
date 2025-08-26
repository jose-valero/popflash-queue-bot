package events

import (
	"reflect"
	"sync"
)

type subscriber func(any)

var (
	mu   sync.RWMutex
	subs = map[string][]subscriber{} // nombre de tipo -> subs
)

func typeNameOf[T any]() string {
	var zero *T
	rt := reflect.TypeOf(zero).Elem() // *T -> T, sin dereferenciar nil
	return rt.PkgPath() + "." + rt.Name()
}

func Subscribe[T any](fn func(T)) func() {
	name := typeNameOf[T]()
	wrapped := func(v any) {
		if ev, ok := v.(T); ok {
			fn(ev)
		}
	}

	mu.Lock()
	subs[name] = append(subs[name], wrapped)
	idx := len(subs[name]) - 1
	mu.Unlock()

	return func() {
		mu.Lock()
		defer mu.Unlock()
		ss := subs[name]
		if idx >= 0 && idx < len(ss) {
			subs[name] = append(ss[:idx], ss[idx+1:]...)
		}
	}
}

func Publish[T any](ev T) {
	name := typeNameOf[T]()
	mu.RLock()
	ss := append([]subscriber(nil), subs[name]...)
	mu.RUnlock()
	for _, s := range ss {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// log.Printf("events: subscriber panic: %v", r)
				}
			}()
			s(ev)
		}()
	}
}
