// Package queue - helpers.go
// Small internal helpers kept separate to keep manager.go focused.
package queue

// snapshot returns a deep copy of the given queue, copying the Players slice.
func snapshot(q *Queue) *Queue {
	cp := *q
	cp.Players = append([]Player(nil), q.Players...)
	return &cp
}

// locatePlayer returns (queueIndex, playerIndex) within qs, or (-1,-1) if absent.
// It is intended to be called under the Manager mutex.
func locatePlayer(qs []*Queue, playerID string) (qi, pi int) {
	for qi, q := range qs {
		for pi, p := range q.Players {
			if p.ID == playerID {
				return qi, pi
			}
		}
	}
	return -1, -1
}
