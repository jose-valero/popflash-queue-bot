// Package queue - rebalance.go
// Queue compaction and housekeeping utilities.
package queue

// rebalanceForward fills earlier queues by pulling head players from later queues.
// Intended to be called under the Manager mutex.
func rebalanceForward(qs []*Queue, fromIdx int) {
	for i := fromIdx; i < len(qs)-1; i++ {
		cur := qs[i]
		next := qs[i+1]
		for len(cur.Players) < cur.Capacity && len(next.Players) > 0 {
			// pop head from next
			head := next.Players[0]
			next.Players = next.Players[1:]
			// push tail into cur
			cur.Players = append(cur.Players, head)
		}
	}
}

// pruneTrailingEmpty removes empty queues from the tail, leaving at least one
// queue if there was at least one non-empty before. Caller must hold the mutex.
func pruneTrailingEmpty(qs []*Queue) []*Queue {
	// keep at least one queue structure alive (UX/UI often expects it)
	if len(qs) == 0 {
		return qs
	}
	last := len(qs) - 1
	for last >= 0 && len(qs[last].Players) == 0 {
		last--
	}
	if last < 0 {
		// all empty -> keep a single empty queue instead of zero?
		// Returning an empty slice matches your current behavior of removing tail empties.
		return []*Queue{}
	}
	return qs[:last+1]
}
