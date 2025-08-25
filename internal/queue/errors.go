// Package queue - errors.go
// Centralized, comparable error values used across the manager logic.
package queue

// qerr is a lightweight comparable error type.
// Using constants of this type allows errors.Is to work as expected.
type qerr string

func (e qerr) Error() string { return string(e) }

var (
	ErrExists    = qerr("queue already exists")
	ErrNotFound  = qerr("queue not found")
	ErrFull      = qerr("queue is full")
	ErrAlreadyIn = qerr("already in queue")
	ErrNotIn     = qerr("player not in queue")
)
