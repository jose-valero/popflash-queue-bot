package queue

import "time"

// represents a player in queue
type Player struct {
	ID       string    // player discord ID
	Username string    // player name
	JoinedAt time.Time // when player join the queue
}

// represents a queue itself

type Queue struct {
	ID        string    // identifyer of queue (exp: "CSPFXCG-1")
	Name      string    // queue name (exp: #queue-1)
	Players   []Player  // list of player in queue
	CreatedAt time.Time // when the queue was created
	Capacity  int       // capacity of queue
}
